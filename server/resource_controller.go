package server

import (
	"reflect"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"mime"
	"net/url"
	"strings"
	"io/ioutil"
	runtime_debug "runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/eug48/fhir/models"
	"github.com/eug48/fhir/models2"
	"github.com/eug48/fhir/search"
)

// ResourceController provides the necessary CRUD handlers for a given resource.
type ResourceController struct {
	Name   string
	DAL    DataAccessLayer
	Config Config
}

// NewResourceController creates a new resource controller for the passed in resource name and the passed in
// DataAccessLayer.
func NewResourceController(name string, dal DataAccessLayer, config Config) *ResourceController {
	return &ResourceController{
		Name:   name,
		DAL:    dal,
		Config: config,
	}
}

func handlePanics(c *gin.Context) {
	if r := recover(); r != nil {
		switch x := r.(type) {
		case *search.Error:
			c.Render(x.HTTPStatus, CustomFhirRenderer{x.OperationOutcome, c})
			return
		case error:
			cause := errors.Cause(x)
			_, isSchemaError := cause.(models2.FhirSchemaError)
			if isSchemaError {
				outcome := models.NewOperationOutcome("fatal", "structure", cause.Error())
				c.Render(http.StatusBadRequest, CustomFhirRenderer{outcome, c})
			} else {
				fmt.Printf("handlePanics: recovered %+v\n", r)
				runtime_debug.PrintStack()

				outcome := models.NewOperationOutcome("fatal", "exception", x.Error())
				c.Render(http.StatusInternalServerError, CustomFhirRenderer{outcome, c})
			}
			return
		default:
			fmt.Printf("handlePanics: recovered %+v\n", r)
			runtime_debug.PrintStack()

			str := fmt.Sprintf("%#v", r)
			outcome := models.NewOperationOutcome("fatal", "exception", str)
			c.Render(http.StatusInternalServerError, CustomFhirRenderer{outcome, c})
			return
		}
	}
}

// IndexHandler handles requests to list resource instances or search for them.
func (rc *ResourceController) IndexHandler(c *gin.Context) {
	defer handlePanics(c)

	rawQuery := c.Request.URL.RawQuery
	if c.Request.Method == "POST" {
		// handle _search (http://hl7.org/fhir/http.html#search)
		// reading urlencoded form values similarly to http/request.go
		ct := c.Request.Header.Get("Content-Type")
		if ct == "" {
			// RFC 2616, section 7.2.1 - empty type SHOULD be treated as application/octet-stream
			ct = "application/octet-stream"
		}
		var err error
		ct, _, err = mime.ParseMediaType(ct)
		if err != nil {
			outcome := models.NewOperationOutcome("fatal", "structure", "failed to parse Content-Type")
			c.Render(http.StatusUnsupportedMediaType, CustomFhirRenderer{outcome, c})
			return
		}
		if ct == "application/x-www-form-urlencoded" {
			bodyBytes, err := ioutil.ReadAll(c.Request.Body)
			if err != nil {
				panic(fmt.Errorf("failed to read POSTed form body: %#v", err))
			}
			rawQuery = string(bodyBytes)
		}
	}


	searchQuery := search.Query{Resource: rc.Name, Query: rawQuery }
	baseURL := responseURL(c.Request, rc.Config, rc.Name)
	bundle, err := rc.DAL.Search(*baseURL, searchQuery)
	if err != nil {
		panic(errors.Wrap(err, "Search failed"))
	}

	c.Set("bundle", bundle)
	c.Set("Resource", rc.Name)
	c.Set("Action", "search")

	c.Render(http.StatusOK, CustomFhirRenderer{bundle, c})
}

// LoadResource uses the resource id in the request to get a resource from the DataAccessLayer and store it in the
// context.
func (rc *ResourceController) LoadResource(c *gin.Context) (resourceId string, resource *models2.Resource, err error) {
	resourceId = c.Param("id")
	resourceVersionId := c.Param("vid")

	if resourceVersionId == "" {
		resource, err = rc.DAL.Get(resourceId, rc.Name)
	} else {
		resource, err = rc.DAL.GetVersion(resourceId, resourceVersionId, rc.Name)
	}
	if err != nil {
		return "", nil, err
	}

	c.Set("Resource", rc.Name)
	return
}

// ShowHandler handles requests to get a particular resource by ID.
func (rc *ResourceController) ShowHandler(c *gin.Context) {
	defer handlePanics(c)
	c.Set("Action", "read")
	resourceId, resource, err := rc.LoadResource(c)
	if err == nil {
		err = setHeaders(c, rc, false, resource, resourceId)
		if err != nil {
			err = errors.Wrap(err, "ShowHandler setHeaders failed")
		}
	}

	switch (err) {
	case nil:
		c.Render(http.StatusOK, CustomFhirRenderer{resource, c})
	case ErrNotFound:
		c.Status(http.StatusNotFound)
	case ErrDeleted:
		c.Status(http.StatusGone)
	default:
		panic(errors.Wrap(err, "LoadResource failed"))
	}
}

func (rc *ResourceController) HistoryHandler(c *gin.Context) {
	defer handlePanics(c)
	c.Set("Action", "history")

	baseURL := responseURL(c.Request, rc.Config, rc.Name)
	resourceId := c.Param("id")
	bundle, err := rc.DAL.History(*baseURL, rc.Name, resourceId)
	if err != nil && err != ErrNotFound {
		panic(errors.Wrap(err, "History failed"))
	}

	if err == ErrNotFound {
		c.Status(http.StatusNotFound)
		return
	}
	c.Render(http.StatusOK, CustomFhirRenderer{bundle, c})
}

// EverythingHandler handles requests for everything related to a Patient or Encounter resource.
func (rc *ResourceController) EverythingHandler(c *gin.Context) {
	defer handlePanics(c)

	// For now we interpret $everything as the union of _include and _revinclude
	query := fmt.Sprintf("_id=%s&_include=*&_revinclude=*", c.Param("id"))

	searchQuery := search.Query{Resource: rc.Name, Query: query}
	baseURL := responseURL(c.Request, rc.Config, rc.Name)
	bundle, err := rc.DAL.Search(*baseURL, searchQuery)
	if err != nil {
		panic(errors.Wrap(err, "Search (everything) failed"))
	}

	c.Set("bundle", bundle)
	c.Set("Resource", rc.Name)
	c.Set("Action", "search")

	c.Render(http.StatusOK, CustomFhirRenderer{bundle, c})
}

// CreateHandler handles requests to create a new resource instance, assigning it a new ID.
func (rc *ResourceController) CreateHandler(c *gin.Context) {
	defer handlePanics(c)
	resource, err := FHIRBind(c, rc.Config.ValidatorURL)
	if err != nil {
		oo := models.NewOperationOutcome("fatal", "structure", err.Error())
		c.Render(http.StatusBadRequest, CustomFhirRenderer{oo, c})
		return
	}

	// check for conditional create
	ifNoneExist := c.GetHeader("If-None-Exist")
	var httpStatus int
	var resourceId string
	if len(ifNoneExist) > 0 {
		query := search.Query{Resource: rc.Name, Query: ifNoneExist}
		httpStatus, resourceId, resource, err = rc.DAL.ConditionalPost(query, resource)
	} else {
		httpStatus = http.StatusCreated
		resourceId, err = rc.DAL.Post(resource)
	}
	if err != nil {
		panic(errors.Wrap(err, "CreateHandler Post/ConditionalPost failed"))
	}

	c.Set(rc.Name, resource)
	c.Set("Resource", rc.Name)
	c.Set("Action", "create")

	if resource != nil { // nil when e.g. HTTP status from ConditionalPost 412
		err = setHeaders(c, rc, true, resource, resourceId)
		if err != nil {
			panic(errors.Wrap(err, "CreateHandler setHeaders failed"))
		}
	}

	c.Render(httpStatus, CustomFhirRenderer{resource, c})
}

// UpdateHandler handles requests to update a resource having a given ID.  If the resource with that ID does not
// exist, a new resource is created with that ID.
func (rc *ResourceController) UpdateHandler(c *gin.Context) {
	defer handlePanics(c)
	resource, err := FHIRBind(c, rc.Config.ValidatorURL)
	if err != nil {
		oo := models.NewOperationOutcome("fatal", "structure", err.Error())
		c.Render(http.StatusBadRequest, CustomFhirRenderer{oo, c})
		return
	}

	resourceId := c.Param("id")
	createdNew, err := rc.DAL.Put(resourceId, resource)
	if err != nil {
		panic(errors.Wrap(err, "Put failed"))
	}

	c.Set(rc.Name, resource)
	c.Set("Resource", rc.Name)

	// spec implies location header only set when createdNew
	setLocationHeader := createdNew
	err = setHeaders(c, rc, setLocationHeader, resource, resourceId)
	
	if createdNew {
		c.Set("Action", "create")
		c.Render(http.StatusCreated, CustomFhirRenderer{resource, c})
	} else {
		c.Set("Action", "update")
		c.Render(http.StatusOK, CustomFhirRenderer{resource, c})
	}
}

// ConditionalUpdateHandler handles requests for conditional updates.  These requests contain search criteria for the
// resource to update.  If the criteria results in no found resources, a new resource is created.  If the criteria
// results in one found resource, that resource will be updated.  Criteria resulting in more than one found resource
// is considered an error.
func (rc *ResourceController) ConditionalUpdateHandler(c *gin.Context) {
	defer handlePanics(c)
	resource, err := FHIRBind(c, rc.Config.ValidatorURL)
	if err != nil {
		oo := models.NewOperationOutcome("fatal", "structure", err.Error())
		c.Render(http.StatusBadRequest, CustomFhirRenderer{oo, c})
		return
	}

	query := search.Query{Resource: rc.Name, Query: c.Request.URL.RawQuery}
	resourceId, createdNew, err := rc.DAL.ConditionalPut(query, resource)
	if err == ErrMultipleMatches {
		c.AbortWithStatus(http.StatusPreconditionFailed)
		return
	} else if err != nil {
		panic(errors.Wrap(err, "ConditionalPut failed"))
	}

	c.Set("Resource", rc.Name)

	err = setHeaders(c, rc, true, resource, resourceId)

	if createdNew {
		c.Set("Action", "create")
		c.Render(http.StatusCreated, CustomFhirRenderer{resource, c})
	} else {
		c.Set("Action", "update")
		c.Render(http.StatusOK, CustomFhirRenderer{resource, c})
	}
}

// DeleteHandler handles requests to delete a resource instance identified by its ID.
func (rc *ResourceController) DeleteHandler(c *gin.Context) {
	defer handlePanics(c)
	id := c.Param("id")

	newVersionId, err := rc.DAL.Delete(id, rc.Name)
	if err != nil && err != ErrNotFound {
		panic(errors.Wrap(err, "Delete failed"))
		return
	}

	c.Set(rc.Name, id)
	c.Set("Resource", rc.Name)
	c.Set("Action", "delete")

	if newVersionId != "" {
		c.Header("ETag", "W/\"" + newVersionId + "\"")
	}
	c.Status(http.StatusNoContent)
}

// ConditionalDeleteHandler handles requests to delete resources identified by search criteria.  All resources
// matching the search criteria will be deleted.
func (rc *ResourceController) ConditionalDeleteHandler(c *gin.Context) {
	defer handlePanics(c)
	query := search.Query{Resource: rc.Name, Query: c.Request.URL.RawQuery}
	_, err := rc.DAL.ConditionalDelete(query)
	if err != nil {
		panic(errors.Wrap(err, "ConditionalDelete failed"))
	}

	c.Set("Resource", rc.Name)
	c.Set("Action", "delete")

	c.Status(http.StatusNoContent)
}

func responseURL(r *http.Request, config Config, paths ...string) *url.URL {

	if config.ServerURL != "" {
		theURL := fmt.Sprintf("%s/%s", strings.TrimSuffix(config.ServerURL, "/"), strings.Join(paths, "/"))
		responseURL, err := url.Parse(theURL)

		if err == nil {
			return responseURL
		}
	}

	responseURL := url.URL{}

	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		responseURL.Scheme = "https"
	} else {
		responseURL.Scheme = "http"
	}
	responseURL.Host = r.Host
	responseURL.Path = fmt.Sprintf("/%s", strings.Join(paths, "/"))

	return &responseURL
}

func setHeaders(c *gin.Context, rc *ResourceController, setLocationHeader bool, resource *models2.Resource, id string) error {
	lastUpdated := resource.LastUpdated()
	if lastUpdated != "" {
		c.Header("Last-Modified", resource.LastUpdatedTime().UTC().Format(http.TimeFormat))
	}

	versionId := resource.VersionId()
	if versionId != "" {
		c.Header("ETag", "W/\"" + versionId + "\"")
	}

	if setLocationHeader {
		if rc.Config.EnableHistory && versionId != "" {
			c.Header("Location", responseURL(c.Request, rc.Config, rc.Name, id, "_history", versionId).String())
		} else {
			c.Header("Location", responseURL(c.Request, rc.Config, rc.Name, id).String())
		}
	}

	return nil
}

// CustomFhirRenderer replaces gin's default JSON renderer and ensures
// that the special characters "<", ">", and "&" are not escaped after the
// the JSON is marshaled. Escaping these special HTML characters is the default
// behavior of Go's json.Marshal().
// It also outputs XML if that is required
type CustomFhirRenderer struct {
	obj interface{}
	c   *gin.Context
}

var fhirJSONContentType = []string{"application/fhir+json; charset=utf-8"}
var fhirXMLContentType = []string{"application/fhir+xml; charset=utf-8"}

func (u CustomFhirRenderer) Render(w http.ResponseWriter) (err error) {

	objVal := reflect.ValueOf(u.obj)
	if u.obj == nil || (objVal.Kind() == reflect.Ptr && objVal.IsNil()) {
		w.Write([]byte(""))
		return
	}

	// fmt.Printf("[CustomFhirRenderer] obj: %+v\n", u.obj)
	data, err := json.Marshal(&u.obj)
	if err != nil {
		return
	}

	if u.c.GetBool("SendXML") {
		converterInt := u.c.MustGet("FhirFormatConverter")
		converter := converterInt.(*FhirFormatConverter)
		var xml string
		xml, err = converter.JsonToXml(string(data))
		if err != nil {
			err = errors.Wrap(err, "CustomFhirRenderer: JsonToXml failed")
			fmt.Printf("ERROR: JsonToXml failed for data: %+v %s\n", u.obj, string(data))
			return
		}
		writeContentType(w, fhirXMLContentType)
		_, err = w.Write([]byte(xml))
	} else {
		// Replace the escaped characters in the data
		data = bytes.Replace(data, []byte("\\u003c"), []byte("<"), -1)
		data = bytes.Replace(data, []byte("\\u003e"), []byte(">"), -1)
		data = bytes.Replace(data, []byte("\\u0026"), []byte("&"), -1)

		writeContentType(w, fhirJSONContentType)
		_, err = w.Write(data)
	}
	return
}

func (u CustomFhirRenderer) WriteContentType(w http.ResponseWriter) {
	writeContentType(w, fhirJSONContentType)
}

func writeContentType(w http.ResponseWriter, value []string) {
	header := w.Header()
	if val := header["Content-Type"]; len(val) == 0 {
		header["Content-Type"] = value
	}
}

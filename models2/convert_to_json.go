package models2

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/mgo.v2/bson"
)

// Reverses transformations done by ConvertJsonToGoFhirBSON
func ConvertGoFhirBSONToJSON(bsonDoc []bson.DocElem) (jsonBytes []byte, includedDocsJsons [][]byte, err error) {

	debug("=== ConvertGoFhirBSONToJSON ===")
	debug("%+v", bsonDoc)

	includedDocsJsons, err = processIncludedDocuments(bsonDoc)
	if err != nil {
		return nil, nil, errors.Wrap(err, "processIncludedDocuments failed")
	}

	var out bytes.Buffer
	err = processDocument(&out, bsonDoc)
	if err != nil {
		return nil, nil, errors.Wrap(err, "ConvertGoFhirBSONToJSON failed")
	}

	debug("ConvertGoFhirBSONToJSON: done")

	jsonBytes = out.Bytes()
	return
}

func processIncludedDocuments(bsonDoc []bson.DocElem) (includedDocsJsons [][]byte, err error) {

	for _, elem := range bsonDoc {
		if docIncluded(elem.Name) {
			// included by a search with _include or _revinclude

			// includedFieldRegex := regexp.MustCompile(`^_included([[:alpha:]]+)ResourcesReferencedBy([[:alpha:]]+)$`)
			arr := elem.Value.([]interface{})
			for _, elt := range arr {

				var includedDoc bson.D
				switch eltV := elt.(type) {
				case []bson.DocElem:
					includedDoc = eltV
				case bson.D:
					includedDoc = eltV
				}

				jsonBytes, nestedIncluded, err := ConvertGoFhirBSONToJSON(includedDoc)
				if err != nil {
					return nil, errors.Wrapf(err, "processIncludedDocuments: ConvertGoFhirBSONToJSON failed at %s", elem.Name)
				}
				if len(nestedIncluded) > 0 {
					return nil, errors.Wrapf(err, "processIncludedDocuments: unexpected nested _included at %s", elem.Name)
				}
				includedDocsJsons = append(includedDocsJsons, jsonBytes)
			}
		}
	}
	return
}

func processDocument(out *bytes.Buffer, bsonDoc []bson.DocElem) (err error) {

	debug("processDocument")

	// handle { time, precision } sub-docs from previous GoFHIR versions (fhirdatetime.go)
	var oldTime, oldPrecision string
	var otherFields bool

	// for dates and decimal documents, just grab the original string and skip the doc entirely
	for _, elem := range bsonDoc {
		debug("  key: %s --> %T", elem.Name, elem.Value)
		switch elem.Name {
		case Gofhir__strNum:
			val, isString := elem.Value.(string)
			if !isString {
				return fmt.Errorf("Error loading BSON: %s element that is not a string", Gofhir__strNum)
			}
			out.WriteString(val)
			debug("processDocument: handled number")
			return
		case Gofhir__strDate:
			val, isString := elem.Value.(string)
			if !isString {
				return fmt.Errorf("Error loading BSON: %s element that is not a string", Gofhir__strDate)
			}

			var jsonStr []byte
			jsonStr, err = json.Marshal(val)
			if err != nil {
				return err
			}
			out.Write(jsonStr)
			debug("processDocument: handled date")
			return

		case "time":
			val, isTime := elem.Value.(time.Time)
			if isTime {
				var jsonStr []byte
				jsonStr, err = json.Marshal(val)
				if err != nil {
					return errors.Wrap(err, "failed to load legacy time field")
				}
				oldTime = string(jsonStr)
			}
		case "precision":
			val, isString := elem.Value.(string)
			if isString {
				oldPrecision = val
			}

		default:
			otherFields = true
		}
	}

	if oldTime != "" && oldPrecision != "" && otherFields == false {
		// TODO: remove or put behind a flag
		out.WriteString(oldTime)
		debug("processDocument: handled old time/precision values (FIXME)")
		return
	}

	out.WriteString("{ ")

	for i, elem := range bsonDoc {

		debug("processDocument: %s", elem.Name)

		switch elem.Name {
		case "reference__id", "reference__type", "reference__external":
			continue // i.e. skip
		}

		if docIncluded(elem.Name) {
			continue // handled above
		}
		if strings.HasPrefix(elem.Name, "_lookup") {
			continue // handled above
		}

		if i > 0 {
			out.WriteString(", ")
		}

		if elem.Name == "extension" || elem.Name == "modifierExtension" {
			out.WriteRune('"')
			out.WriteString(elem.Name)
			out.WriteString("\": ")
			err := processExtensionsArray(out, elem.Value)
			if err != nil {
				return err
			}
		} else {
			bsonKey := elem.Name
			switch bsonKey {
			case "_id":
				bsonKey = "id"
			case "__id":
				bsonKey = "_id"
			}

			key, err := json.Marshal(bsonKey)
			if err != nil {
				return err
			}
			out.Write(key)
			out.WriteString(": ")
			err = processValue(out, elem.Value)
			if err != nil {
				return err
			}
		}
	}

	out.WriteString(" }")
	return err
}

func processValue(out *bytes.Buffer, elt interface{}) (err error) {

	debug("  processValue: %T", elt)

	switch v := elt.(type) {
	case int, int32, int64, float32, float64:
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		out.Write(b)

	case *int, *int32, *int64, *float32, *float64:
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		out.Write(b)

	case *time.Time, time.Time, string:
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		out.Write(b)

	case bool:
		if v {
			out.WriteString("true")
		} else {
			out.WriteString("false")
		}

	case *[]bson.DocElem:
		err = processDocument(out, *v)

	case []bson.DocElem:
		err = processDocument(out, v)

	case bson.D:
		err = processDocument(out, v)

	case *[]interface{}:
		err = processArray(out, *v)

	case []interface{}:
		err = processArray(out, v)

	case nil:
		out.WriteString("null")

	default:
		strType := reflect.TypeOf(elt).String()
		panic(fmt.Errorf("processValue: unhandled type %s", strType))
	}

	return
}

func processArray(out *bytes.Buffer, v []interface{}) error {
	debug("  processArray: %T", v)
	out.WriteString("[ ")
	for i, elt := range v {

		if elt != nil {
			err := processValue(out, elt)
			if err != nil {
				return err
			}
		} else {
			out.WriteString("null")
		}

		if i < len(v)-1 {
			out.WriteString(", ")
		}
	}
	out.WriteString(" ]")
	return nil
}

func processExtensionsArray(out *bytes.Buffer, value interface{}) error {

	debug("  processExtensionsArray: %+v", value, value)

	var array []interface{}
	// var array []bson.DocElem
	switch v := value.(type) {
	case *[]interface{}:
		array = *v
	case []interface{}:
		array = v
	// case []bson.DocElem:
	// array = v
	default:
		return fmt.Errorf("processExtensionsArray: value of unexpected type %T", value)
	}

	out.WriteString("[ ")
	for i, elt := range array {
		if i > 0 {
			out.WriteString(", ")
		}
		debug("  processExtensionsArray: elt %+v", elt)

		// subdoc, ok := elt.Value.([]bson.DocElem)
		var subdoc bson.D
		switch eltV := elt.(type) {
		case []bson.DocElem:
			subdoc = eltV
		case bson.D:
			subdoc = eltV
		default:
			return fmt.Errorf("processExtensionsArray: element of unexpected type %T", elt)
		}
		if len(subdoc) != 1 {
			return fmt.Errorf("processExtensionsArray: element of unexpected length %d", len(subdoc))
		}
		subdoc1 := subdoc[0]
		url := subdoc1.Name

		var subsubdoc bson.D
		switch eltV := subdoc1.Value.(type) {
		case []bson.DocElem:
			subsubdoc = eltV
		case bson.D:
			subsubdoc = eltV
		default:
			return fmt.Errorf("processExtensionsArray: sub-document of unexpected type %T", subdoc1.Value)
		}

		originalExtension := make([]bson.DocElem, 0, 1+len(subsubdoc))
		originalExtension = append(originalExtension, bson.DocElem{Name: "url", Value: url})
		originalExtension = append(originalExtension, subsubdoc...)

		err := processDocument(out, originalExtension)
		if err != nil {
			return err
		}
	}
	out.WriteString("] ")

	return nil
}

func docIncluded(key string) bool {
	return strings.HasPrefix(key, "_included") || strings.HasPrefix(key, "_revIncluded")
}

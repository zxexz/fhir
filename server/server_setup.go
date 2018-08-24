package server

import (
	"github.com/pkg/errors"
	"net/url"
	"fmt"
	"log"
	"time"
	"context"

	"github.com/gin-gonic/gin"
	"github.com/itsjamie/gin-cors"
	"github.com/mongodb/mongo-go-driver/mongo"
)

type AfterRoutes func(*gin.Engine)

type FHIRServer struct {
	Config           Config
	Engine           *gin.Engine
	MiddlewareConfig map[string][]gin.HandlerFunc
	AfterRoutes      []AfterRoutes
	Interceptors     map[string]InterceptorList
}

func (f *FHIRServer) AddMiddleware(key string, middleware gin.HandlerFunc) {
	f.MiddlewareConfig[key] = append(f.MiddlewareConfig[key], middleware)
}

// AddInterceptor adds a new interceptor for a particular database operation and FHIR resource.
// For example:
// AddInterceptor("Create", "Patient", patientInterceptorHandler) would register the
// patientInterceptorHandler methods to be run against a Patient resource when it is created.
//
// To run a handler against ALL resources pass "*" as the resourceType.
//
// Supported database operations are: "Create", "Update", "Delete"
func (f *FHIRServer) AddInterceptor(op, resourceType string, handler InterceptorHandler) error {

	if op == "Create" || op == "Update" || op == "Delete" {
		f.Interceptors[op] = append(f.Interceptors[op], Interceptor{ResourceType: resourceType, Handler: handler})
		return nil
	}
	return fmt.Errorf("AddInterceptor: unsupported database operation %s", op)
}

func NewServer(config Config) *FHIRServer {
	server := &FHIRServer{
		Config:           config,
		MiddlewareConfig: make(map[string][]gin.HandlerFunc),
		Interceptors:     make(map[string]InterceptorList),
	}
	server.Engine = gin.Default()

	if config.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	gin.DisableConsoleColor()

	server.Engine.Use(cors.Middleware(cors.Config{
		Origins:         "*",
		Methods:         "GET, PUT, POST, DELETE",
		RequestHeaders:  "Origin, Authorization, Content-Type, If-Match, If-None-Exist",
		ExposedHeaders:  "Location, ETag, Last-Modified",
		MaxAge:          86400 * time.Second, // Preflight expires after 1 day
		Credentials:     true,
		ValidateHeaders: false,
	}))

	if config.EnableXML {
		server.Engine.Use(EnableXmlToJsonConversionMiddleware())
		server.Engine.Use(AbortNonFhirXMLorJSONRequestsMiddleware)
	} else {
		server.Engine.Use(AbortNonJSONRequestsMiddleware)
	}

	if config.ReadOnly {
		server.Engine.Use(ReadOnlyMiddleware)
	}

	return server
}

func (f *FHIRServer) Run() {
	var err error

	// Establish initial connection to mongo
	client, err := mongo.Connect(context.Background(), fmt.Sprintf("mongodb://%s", f.Config.DatabaseHost))
	if err != nil {
		panic(errors.Wrap(err, "connecting to MongoDB"))
	}

	// session.SetSafe(&mgo.Safe{}) // makes the session check for errors such as ErrNotFound

	log.Println("MongoDB: Connected")

	// Establish fhir database session
	masterSession := NewMasterSession(client, f.Config.DatabaseName)

	// Ensure all indexes
	NewIndexer(f.Config).ConfigureIndexes(masterSession)

	// Establish admin session
	masterAdminSession := NewMasterSession(client, "admin")

	// Kick off the database op monitoring routine. This periodically checks db.currentOp() and
	// kills client-initiated operations exceeding the configurable timeout. Do this AFTER the index
	// build to ensure no index build processes are killed unintentionally.
	ticker := time.NewTicker(f.Config.DatabaseKillOpPeriod)
	go killLongRunningOps(ticker, masterAdminSession, f.Config)

	// Register all API routes
	RegisterRoutes(f.Engine, f.MiddlewareConfig, NewMongoDataAccessLayer(masterSession, f.Interceptors, f.Config), f.Config)

	for _, ar := range f.AfterRoutes {
		ar(f.Engine)
	}

	// If not in -readonly mode, clear the count cache
	if !f.Config.ReadOnly {
		worker := masterSession.GetWorkerSession()
		defer worker.Close()
		count, err := worker.DB().Collection("countcache").Count(context.Background(), nil)
		if count > 0 || err != nil {
			err = worker.DB().Collection("countcache").Drop(context.Background())
			if err != nil {
				panic(fmt.Sprintf("Server: Failed to clear count cache (%+v)", err))
			}
		}
	} else {
		log.Println("Server: Running in read-only mode")
	}


	url, err := url.Parse(f.Config.ServerURL)
	if err != nil {
		panic("Server: Failed to parse ServerURL: " + f.Config.ServerURL)
	}
	f.Engine.Run(":" + url.Port())
}

package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	appconfig "github.com/workshop/wrong-question/internal/config"
	"github.com/workshop/wrong-question/internal/db"
	"github.com/workshop/wrong-question/internal/handler"
	"github.com/workshop/wrong-question/internal/middleware"
	"github.com/workshop/wrong-question/internal/repository"
	"github.com/workshop/wrong-question/internal/service"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg, err := appconfig.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()

	mysqlDB, err := db.NewMySQL(ctx, cfg)
	if err != nil {
		slog.Error("connect mysql", "error", err)
		os.Exit(1)
	}
	defer mysqlDB.Close()

	ddbClient, err := db.NewDynamoDB(ctx, cfg.AWSRegion)
	if err != nil {
		slog.Error("connect dynamodb", "error", err)
		os.Exit(1)
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(cfg.AWSRegion))
	if err != nil {
		slog.Error("load aws config", "error", err)
		os.Exit(1)
	}
	bedrockClient := bedrockruntime.NewFromConfig(awsCfg)
	cognitoClient := cognitoidentityprovider.NewFromConfig(awsCfg)

	// repositories
	userRepo := repository.NewUserRepo(mysqlDB)
	questionRepo := repository.NewQuestionRepo(mysqlDB)
	errorRepo := repository.NewErrorRecordRepo(mysqlDB)
	reviewRepo := repository.NewReviewRepo(mysqlDB, ddbClient, cfg.DynamoTableSchedule)

	// services
	recognitionSvc := service.NewRecognitionService(cfg.RecognitionAPIURL, cfg.RecognitionAPIKey, cfg.ExternalTimeoutSec)
	bedrockSvc := service.NewBedrockService(bedrockClient, cfg.BedrockModelID, cfg.ExternalTimeoutSec)
	questionSvc := service.NewQuestionService(questionRepo, recognitionSvc, cfg.ImageDir)
	reviewSvc := service.NewReviewService(reviewRepo)
	exportSvc := service.NewExportService(questionRepo, cfg.ImageDir+"/exports")

	// handlers
	authH := handler.NewAuthHandler(cognitoClient, cfg.CognitoClientID, cfg.CognitoUserPoolID, userRepo)
	questionH := handler.NewQuestionHandler(questionSvc)
	reviewH := handler.NewReviewHandler(reviewSvc)
	recommendH := handler.NewRecommendHandler(bedrockSvc, errorRepo)
	exportH := handler.NewExportHandler(exportSvc)

	r := chi.NewRouter()
	r.Use(middleware.Recovery)
	r.Use(middleware.Logger)
	r.Use(chimiddleware.RealIP)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: false,
	}))

	// public routes
	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/register", authH.Register)
		r.Post("/login", authH.Login)
		r.Post("/refresh", authH.Refresh)
	})

	// protected routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.CognitoJWTAuth(cfg.AWSRegion, cfg.CognitoUserPoolID))

		r.Route("/api/questions", func(r chi.Router) {
			r.Post("/", questionH.Upload)
			r.Get("/", questionH.List)
			r.Get("/{id}", questionH.Get)
			r.Delete("/{id}", questionH.Delete)
		})

		r.Route("/api/review", func(r chi.Router) {
			r.Get("/today", reviewH.Today)
			r.Post("/{id}/result", reviewH.SubmitResult)
		})

		r.Get("/api/recommend", recommendH.Recommend)
		r.Post("/api/export/pdf", exportH.ExportPDF)
	})

	// serve uploaded images
	r.Handle("/images/*", http.StripPrefix("/images/", http.FileServer(http.Dir(cfg.ImageDir))))

	slog.Info("server starting", "port", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

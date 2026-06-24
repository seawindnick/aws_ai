package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3vectors"
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

	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(cfg.AWSRegion))
	if err != nil {
		slog.Error("load aws config", "error", err)
		os.Exit(1)
	}

	ddbClient := dynamodb.NewFromConfig(awsCfg)
	bedrockClient := bedrockruntime.NewFromConfig(awsCfg)
	cognitoClient := cognitoidentityprovider.NewFromConfig(awsCfg)
	s3vClient := s3vectors.NewFromConfig(awsCfg)

	// repositories
	userRepo := repository.NewUserRepo(mysqlDB)
	questionRepo := repository.NewQuestionRepo(mysqlDB)
	tagRepo := repository.NewTagRepo(mysqlDB)
	paperRepo := repository.NewPaperRepo(mysqlDB)
	notifRepo := repository.NewNotificationRepo(mysqlDB)
	errorRepo := repository.NewErrorRecordRepo(mysqlDB)
	reviewRepo := repository.NewReviewRepo(mysqlDB, ddbClient, cfg.DynamoTableSchedule)
	classRepo := repository.NewClassRepo(mysqlDB)
	taskRepo := repository.NewTaskRepo(mysqlDB)
	s3vRepo := repository.NewS3VectorsRepo(s3vClient, cfg.S3VectorsBucket)

	// services
	recognitionSvc := service.NewRecognitionService(cfg.RecognitionAPIURL, cfg.RecognitionAPIKey, cfg.ExternalTimeoutSec)
	bedrockSvc := service.NewBedrockService(bedrockClient, cfg.BedrockModelID, cfg.ExternalTimeoutSec)
	embedSvc := service.NewEmbeddingService(bedrockClient, cfg.EmbeddingModelID, cfg.ExternalTimeoutSec)
	embedJobSvc := service.NewEmbeddingJobService(ddbClient, cfg.DynamoTableEmbedJobs)
	notifSvc := service.NewNotificationService(notifRepo)
	questionSvc := service.NewQuestionService(questionRepo, tagRepo, recognitionSvc, embedJobSvc, cfg)
	tagSvc := service.NewTagService(tagRepo, questionRepo)
	paperSvc := service.NewPaperService(paperRepo, questionRepo)
	reviewSvc := service.NewReviewService(reviewRepo, errorRepo)
	reviewQueueSvc := service.NewReviewQueueService(questionRepo, notifSvc)
	exportSvc := service.NewExportService(questionRepo, paperRepo, cfg.ExportDir)
	meSvc := service.NewMeService(userRepo, cognitoClient, cfg.CognitoClientID, cfg.CognitoUserPoolID)
	adminSvc := service.NewAdminService(userRepo, cognitoClient, cfg.CognitoClientID, cfg.CognitoUserPoolID)
	statsSvc := service.NewStatsService(questionRepo, reviewRepo, userRepo)
	classSvc := service.NewClassService(classRepo)
	taskSvc := service.NewTaskService(taskRepo, classRepo, reviewSvc)
	semanticSvc := service.NewSemanticSearchService(embedSvc, s3vRepo, questionRepo)

	// handlers
	authH := handler.NewAuthHandler(cognitoClient, cfg.CognitoClientID, cfg.CognitoUserPoolID, userRepo)
	questionH := handler.NewQuestionHandler(questionSvc)
	tagH := handler.NewTagHandler(tagSvc)
	paperH := handler.NewPaperHandler(paperSvc, exportSvc)
	notifH := handler.NewNotificationHandler(notifSvc)
	reviewH := handler.NewReviewHandler(reviewSvc)
	reviewQueueH := handler.NewReviewQueueHandler(reviewQueueSvc)
	recommendH := handler.NewRecommendHandler(bedrockSvc, errorRepo)
	exportH := handler.NewExportHandler(exportSvc)
	meH := handler.NewMeHandler(meSvc)
	adminH := handler.NewAdminHandler(adminSvc, questionSvc)
	statsH := handler.NewStatsHandler(statsSvc)
	classH := handler.NewClassHandler(classSvc)
	taskH := handler.NewTaskHandler(taskSvc)
	errorRecordH := handler.NewErrorRecordHandler(errorRepo)
	semanticH := handler.NewSemanticSearchHandler(semanticSvc)

	r := chi.NewRouter()
	r.Use(middleware.Recovery)
	r.Use(middleware.Logger)
	r.Use(chimiddleware.RealIP)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: false,
	}))

	// health check
	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// public routes
	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/login", authH.Login)
		r.Post("/refresh", authH.Refresh)
	})

	// protected routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.CognitoJWTAuth(cfg.AWSRegion, cfg.CognitoUserPoolID, userRepo))

		// profile
		r.Route("/api/me", func(r chi.Router) {
			r.Get("/", meH.GetProfile)
			r.Post("/nickname", meH.UpdateNickname)
			r.Post("/password", meH.ChangePassword)
			r.Post("/deactivate", meH.Deactivate)
		})

		// admin-only
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireRole("admin"))
			r.Route("/api/admin", func(r chi.Router) {
				r.Post("/users", adminH.CreateUser)
				r.Post("/users/import", adminH.ImportCSV)
				r.Get("/users", adminH.ListUsers)
				r.Post("/users/status", adminH.SetStatus)
				r.Post("/users/role", adminH.SetRole)
				r.Get("/users/questions", adminH.ListUserQuestions)
			})
		})

		// questions
		r.Route("/api/questions", func(r chi.Router) {
			r.Post("/", questionH.Upload)
			r.Post("/batch", questionH.BatchUpload)
			r.Get("/", questionH.List)
			r.Get("/search", questionH.Search)
			r.Get("/semantic-search", semanticH.Search)
			r.Get("/{id}", questionH.Get)
			r.Delete("/{id}", questionH.Delete)
			r.Patch("/{id}/category", questionH.UpdateCategory)
			// tags under question
			r.Get("/{id}/tags", tagH.List)
			r.Post("/{id}/tags", tagH.AddManual)
			r.Post("/{id}/tags/{tid}/confirm", tagH.Confirm)
			r.Delete("/{id}/tags/{tid}", tagH.Delete)
		})

		// papers
		r.Route("/api/papers", func(r chi.Router) {
			r.Post("/", paperH.Create)
			r.Get("/", paperH.List)
			r.Get("/download", paperH.Download)
			r.Get("/{id}", paperH.Get)
			r.Patch("/{id}", paperH.Rename)
			r.Delete("/{id}", paperH.Delete)
			r.Post("/{id}/questions", paperH.AddQuestion)
			r.Delete("/{id}/questions/{qid}", paperH.RemoveQuestion)
			r.Put("/{id}/reorder", paperH.Reorder)
			r.Get("/{id}/questions", paperH.ListQuestions)
			r.Post("/{id}/duplicate", paperH.Duplicate)
			r.Post("/{id}/export", paperH.ExportPDF)
		})

		// notifications
		r.Route("/api/notifications", func(r chi.Router) {
			r.Get("/", notifH.List)
			r.Post("/read-all", notifH.MarkAllRead)
			r.Post("/{id}/read", notifH.MarkRead)
		})

		// review queue: teacher/admin only
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireRole("teacher", "admin"))
			r.Get("/api/review-queue", reviewQueueH.ListPending)
			r.Post("/api/review-queue/{id}/review", reviewQueueH.Review)
		})

		// spaced review
		r.Route("/api/review", func(r chi.Router) {
			r.Get("/today", reviewH.Today)
			r.Post("/{id}/result", reviewH.SubmitResult)
		})

		// error records
		r.Route("/api/error-records", func(r chi.Router) {
			r.Get("/", errorRecordH.List)
		})

		// stats
		r.Route("/api/stats", func(r chi.Router) {
			r.Get("/me", statsH.MyStats)
			// teacher/admin only
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireRole("teacher", "admin"))
				r.Get("/class", statsH.ClassStats)
				r.Get("/student", statsH.StudentStats)
			})
		})

		// classes
		r.Route("/api/classes", func(r chi.Router) {
			r.Post("/", classH.Create)
			r.Get("/", classH.List)
			r.Get("/detail", classH.Detail)
			r.Post("/join", classH.Join)
			r.Post("/leave", classH.Leave)
			r.Post("/remove-member", classH.RemoveMember)
			r.Post("/reset-code", classH.ResetCode)
		})

		// tasks
		r.Route("/api/tasks", func(r chi.Router) {
			r.Post("/", taskH.Create)
			r.Get("/", taskH.List)
			r.Get("/detail", taskH.Detail)
			r.Post("/update", taskH.Update)
			r.Post("/submit", taskH.Submit)
			r.Get("/progress", taskH.Progress)
		})

		// AI recommendations and bulk export
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

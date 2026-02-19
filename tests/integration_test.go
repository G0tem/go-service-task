// Запуск: go test -v -tags=integration ./tests/ -run Integration

//go:build integration

package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	redisv8 "github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
	tcmysql "github.com/testcontainers/testcontainers-go/modules/mysql"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/G0tem/go-service-task/internal/config"
	"github.com/G0tem/go-service-task/internal/handler"
	"github.com/G0tem/go-service-task/internal/model"
	"github.com/G0tem/go-service-task/internal/router"
)

const testSecretKey = "test-secret-key-for-jwt"

func setupMySQL(t *testing.T) (dsn string, cleanup func()) {
	t.Helper()
	ctx := context.Background()
	container, err := tcmysql.Run(ctx, "mysql:8.0",
		tcmysql.WithDatabase("tasktest"),
		tcmysql.WithUsername("user"),
		tcmysql.WithPassword("pass"),
	)
	if err != nil {
		t.Fatalf("failed to start mysql: %v", err)
	}
	dsn, err = container.ConnectionString(ctx, "parseTime=true", "charset=utf8mb4")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}
	cleanup = func() {
		_ = container.Terminate(ctx)
	}
	return dsn, cleanup
}

func setupRedis(t *testing.T) (*redisv8.Client, func()) {
	t.Helper()
	ctx := context.Background()

	redisContainer, err := tcredis.Run(ctx, "redis:7-alpine")
	if err != nil {
		t.Fatalf("failed to start redis container: %v", err)
	}

	host, err := redisContainer.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get redis host: %v", err)
	}
	port, err := redisContainer.MappedPort(ctx, "6379/tcp")
	if err != nil {
		t.Fatalf("failed to get redis port: %v", err)
	}

	addr := fmt.Sprintf("%s:%s", host, port.Port())

	rdb := redisv8.NewClient(&redisv8.Options{
		Addr: addr,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Fatalf("failed to ping redis: %v", err)
	}

	cleanup := func() {
		_ = rdb.Close()
		_ = redisContainer.Terminate(ctx)
	}

	return rdb, cleanup
}

func TestIntegration_RegisterLoginCreateTeamAndTasks(t *testing.T) {
	dsn, cleanupDB := setupMySQL(t)
	defer cleanupDB()

	rdb, cleanupRedis := setupRedis(t)
	defer cleanupRedis()

	// Инициализация GORM (используем алиас gormmysql)
	db, err := gorm.Open(gormmysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Team{}, &model.TeamMember{}, &model.Task{}, &model.TaskHistory{}, &model.TaskComment{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	cfg := &config.Config{
		SecretKey: testSecretKey,
		HttpPort:  8001,
	}
	app := fiber.New()
	h := handler.NewHandler(db, rdb, cfg)
	router.SetupRoutes(app)
	h.SetupRoutes(app)

	// 1) Регистрация
	regBody := map[string]string{
		"email":           "user@test.com",
		"username":        "user1",
		"password":        "password123",
		"confirmPassword": "password123",
	}
	body, _ := json.Marshal(regBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("register request: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register status = %d", resp.StatusCode)
	}
	var regResp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&regResp); err != nil {
		t.Fatalf("decode register: %v", err)
	}
	token := regResp.Data.Token
	if token == "" {
		t.Fatal("empty token")
	}

	// 2) Создание команды
	teamBody := map[string]string{"name": "Team1"}
	body, _ = json.Marshal(teamBody)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/teams", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("create team: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create team status = %d", resp.StatusCode)
	}
	var teamResp struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&teamResp); err != nil {
		t.Fatalf("decode team: %v", err)
	}
	teamID := teamResp.ID
	if teamID == "" {
		t.Fatal("empty team id")
	}

	// 3) Создание нескольких задач
	for i := 0; i < 5; i++ {
		taskBody := map[string]string{
			"title":       fmt.Sprintf("Task %d", i+1),
			"description": "desc",
			"team_id":     teamID,
		}
		body, _ = json.Marshal(taskBody)
		req = httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err = app.Test(req)
		if err != nil {
			t.Fatalf("create task: %v", err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create task status = %d", resp.StatusCode)
		}
	}

	// 4) Список задач с пагинацией
	req = httptest.NewRequest(http.MethodGet, "/api/v1/tasks?team_id="+teamID+"&page=1&page_size=2", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list tasks status = %d", resp.StatusCode)
	}
	var listResp struct {
		Tasks []struct {
			ID string `json:"id"`
		} `json:"tasks"`
		CurrentPage  int   `json:"current_page"`
		PageSize     int   `json:"page_size"`
		TotalPages   int   `json:"total_pages"`
		TotalRecords int64 `json:"total_records"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if listResp.CurrentPage != 1 || listResp.PageSize != 2 {
		t.Errorf("page/size = %d/%d", listResp.CurrentPage, listResp.PageSize)
	}
	if listResp.TotalRecords != 5 {
		t.Errorf("total_records = %d, want 5", listResp.TotalRecords)
	}
	if len(listResp.Tasks) != 2 {
		t.Errorf("len(tasks) = %d, want 2", len(listResp.Tasks))
	}
	if listResp.TotalPages != 3 {
		t.Errorf("total_pages = %d, want 3", listResp.TotalPages)
	}

	// 5) Вторая страница
	req = httptest.NewRequest(http.MethodGet, "/api/v1/tasks?team_id="+teamID+"&page=2&page_size=2", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("list page 2: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list page 2 status = %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode list page 2: %v", err)
	}
	if listResp.CurrentPage != 2 || len(listResp.Tasks) != 2 {
		t.Errorf("page 2: current=%d len(tasks)=%d", listResp.CurrentPage, len(listResp.Tasks))
	}

	// 6) Третья страница
	req = httptest.NewRequest(http.MethodGet, "/api/v1/tasks?team_id="+teamID+"&page=3&page_size=2", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("list page 3: %v", err)
	}
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode list page 3: %v", err)
	}
	if len(listResp.Tasks) != 1 {
		t.Errorf("page 3 len(tasks) = %d, want 1", len(listResp.Tasks))
	}
}

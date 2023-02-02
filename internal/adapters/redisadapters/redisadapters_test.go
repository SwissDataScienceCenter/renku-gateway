package redisadapters

import (
	"context"
	"encoding/json"
	"math/rand"
	"testing"
	"time"

	"github.com/SwissDataScienceCenter/renku-gateway-v2/internal/models"
	"github.com/go-redis/redis/v9"
	"github.com/go-redis/redismock/v9"
)

func TestSetSession(t *testing.T) {
	ctx := context.Background()

	client, mock := redismock.NewClientMock()

	adapter1 := RedisAdapter{
		Rdb: *client,
	}

	expirationTime := time.Unix(time.Now().Unix()+rand.Int63n(14400), 0)
	testTokenIDs := []string{"test"}
	jsonTestTokenIDs, _ := json.Marshal(testTokenIDs)

	mySession := models.Session{
		ID:        "12345",
		Type:      "user",
		ExpiresAt: expirationTime,
		TokenIDs:  testTokenIDs,
	}

	mock.ExpectHSet("session-12345", "type", "user", "expiresAt", expirationTime.Unix(), "tokenIds", jsonTestTokenIDs)

	adapter1.SetSession(ctx, mySession)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGetSession(t *testing.T) {
	ctx := context.Background()

	client, mock := redismock.NewClientMock()

	adapter1 := RedisAdapter{
		Rdb: *client,
	}

	mock.ExpectHGetAll("session-12345")

	adapter1.GetSession(ctx, "12345")

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRemoveSession(t *testing.T) {
	ctx := context.Background()

	client, mock := redismock.NewClientMock()

	adapter1 := RedisAdapter{
		Rdb: *client,
	}

	mock.ExpectDel("session-12345")

	adapter1.RemoveSession(ctx, "12345")

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSetAccessToken(t *testing.T) {
	ctx := context.Background()

	client, mock := redismock.NewClientMock()

	adapter1 := RedisAdapter{
		Rdb: *client,
	}

	expirationTime := time.Unix(time.Now().Unix()+rand.Int63n(14400), 0)

	myAccessToken := models.AccessToken{
		ID:        "12345",
		Value:     "6789",
		ExpiresAt: expirationTime,
		URL:       "https://gitlab.com",
		Type:      "git",
	}

	z1 := redis.Z{
		Score:  float64(myAccessToken.ExpiresAt.Unix()),
		Member: myAccessToken.ID,
	}

	mock.ExpectZAdd("indexExpiringTokens", z1)

	//mock.ExpectHSet("accessTokens-12345", "accessToken", "6789", "expiresAt", expirationTime.Unix(), "URL", "https://gitlab.com", "type", "git")

	adapter1.SetAccessToken(ctx, myAccessToken)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGetAccessToken(t *testing.T) {
	ctx := context.Background()

	client, mock := redismock.NewClientMock()

	adapter1 := RedisAdapter{
		Rdb: *client,
	}

	mock.ExpectHGetAll("accessTokens-12345")

	adapter1.GetAccessToken(ctx, "12345")

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRemoveAccessToken(t *testing.T) {
	ctx := context.Background()

	client, mock := redismock.NewClientMock()

	adapter1 := RedisAdapter{
		Rdb: *client,
	}

	expirationTime := time.Unix(time.Now().Unix()+rand.Int63n(14400), 0)

	myAccessToken := models.AccessToken{
		ID:        "12345",
		Value:     "6789",
		ExpiresAt: expirationTime,
		URL:       "https://gitlab.com",
		Type:      "git",
	}

	z1 := redis.Z{
		Score:  float64(myAccessToken.ExpiresAt.Unix()),
		Member: myAccessToken.ID,
	}

	mock.ExpectZRem("indexExpiringTokens", z1)

	//mock.ExpectDel("accessTokens-12345")

	adapter1.RemoveAccessToken(ctx, myAccessToken)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSetRefreshToken(t *testing.T) {
	ctx := context.Background()

	client, mock := redismock.NewClientMock()

	adapter1 := RedisAdapter{
		Rdb: *client,
	}

	expirationTime := time.Unix(time.Now().Unix()+rand.Int63n(14400), 0)

	myRefreshToken := models.RefreshToken{
		ID:        "12345",
		Value:     "6789",
		ExpiresAt: expirationTime,
	}

	mock.ExpectHSet("refreshTokens-12345", "refreshToken", "6789", "expiresAt", expirationTime.Unix())

	adapter1.SetRefreshToken(ctx, myRefreshToken)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGetRefreshToken(t *testing.T) {
	ctx := context.Background()

	client, mock := redismock.NewClientMock()

	adapter1 := RedisAdapter{
		Rdb: *client,
	}

	mock.ExpectHGetAll("refreshTokens-12345")

	adapter1.GetRefreshToken(ctx, "12345")

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRemoveRefreshToken(t *testing.T) {
	ctx := context.Background()

	client, mock := redismock.NewClientMock()

	adapter1 := RedisAdapter{
		Rdb: *client,
	}

	mock.ExpectDel("refreshTokens-12345")

	adapter1.RemoveRefreshToken(ctx, "12345")

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGetExpiringAccessTokenIDs(t *testing.T) {
	ctx := context.Background()

	client, mock := redismock.NewClientMock()

	adapter1 := RedisAdapter{
		Rdb: *client,
	}

	startTime := time.Now()

	stopTime := time.Now().Add(time.Hour * 4)

	zRangeArgs := redis.ZRangeArgs{
		Key:     "indexExpiringTokens",
		Start:   startTime.Unix(),
		Stop:    stopTime.Unix(),
		ByScore: true,
	}

	mock.ExpectZRangeArgsWithScores(zRangeArgs)

	adapter1.GetExpiringAccessTokenIDs(ctx, startTime, stopTime)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSetProjectToken(t *testing.T) {
	ctx := context.Background()

	client, mock := redismock.NewClientMock()

	adapter1 := RedisAdapter{
		Rdb: *client,
	}

	expirationTime := time.Unix(time.Now().Unix()+rand.Int63n(14400), 0)

	myAccessToken := models.AccessToken{
		ID:        "12345",
		Value:     "6789",
		ExpiresAt: expirationTime,
		URL:       "https://gitlab.com",
		Type:      "git",
	}

	z1 := redis.Z{
		Score:  float64(myAccessToken.ExpiresAt.Unix()),
		Member: myAccessToken.ID,
	}

	mock.ExpectZAdd("projectTokens-4567", z1)

	adapter1.SetProjectToken(ctx, 4567, myAccessToken)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGetProjectTokens(t *testing.T) {
	ctx := context.Background()

	client, mock := redismock.NewClientMock()

	adapter1 := RedisAdapter{
		Rdb: *client,
	}

	zRangeArgs := redis.ZRangeArgs{
		Key:     "projectTokens-4567",
		Start:   0,
		Stop:    999999,
		ByScore: false,
	}

	mock.ExpectZRangeArgsWithScores(zRangeArgs)

	adapter1.GetProjectTokens(ctx, 4567)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRemoveProjectToken(t *testing.T) {
	ctx := context.Background()

	client, mock := redismock.NewClientMock()

	adapter1 := RedisAdapter{
		Rdb: *client,
	}

	expirationTime := time.Unix(time.Now().Unix()+rand.Int63n(14400), 0)

	myAccessToken := models.AccessToken{
		ID:        "12345",
		Value:     "6789",
		ExpiresAt: expirationTime,
		URL:       "https://gitlab.com",
		Type:      "git",
	}

	z1 := redis.Z{
		Score:  float64(myAccessToken.ExpiresAt.Unix()),
		Member: myAccessToken.ID,
	}

	mock.ExpectZRem("projectTokens-4567", z1)

	adapter1.RemoveProjectToken(ctx, 4567, myAccessToken)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

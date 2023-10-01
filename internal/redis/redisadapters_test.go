package redisadapters

import (
	"context"
	"encoding"
	"fmt"
	"math/rand"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/SwissDataScienceCenter/renku-gateway-v2/internal/models"
	"github.com/go-redis/redis/v9"
	"github.com/go-redis/redismock/v9"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
)

var compareOptions []cmp.Option = []cmp.Option{cmpopts.IgnoreUnexported(models.OauthToken{})}

func decomposeStructToMap(strct interface{}) (map[string]string, error) {
	v := reflect.ValueOf(strct)
	t := v.Type()
	output := map[string]string{}
	for i := 0; i < v.NumField(); i++ {
		if !t.Field(i).IsExported() {
			continue
		}
		fieldName := t.Field(i).Name
		fieldValue := v.Field(i).Interface()
		switch fieldValue.(type) {
		case int:
			output[fieldName] = strconv.Itoa(fieldValue.(int))
		case string:
			output[fieldName] = fieldValue.(string)
		default:
			fieldValueEncodable, ok := fieldValue.(encoding.TextMarshaler)
			if !ok {
				return output, fmt.Errorf("value %v must implement encoding.TextMarshaler", fieldValue)
			}
			rawBytes, err := fieldValueEncodable.MarshalText()
			if err != nil {
				return output, err
			}
			output[fieldName] = string(rawBytes)
		}
	}
	return output, nil
}

func TestSetSession(t *testing.T) {
	ctx := context.Background()

	client, mock := redismock.NewClientMock()

	adapter := RedisAdapter{
		rdb: client,
	}

	mySession := models.Session{
		ID:        "12345",
		ExpiresAt: time.Now().Add(time.Second * time.Duration(uint64(rand.Int63n(14400)))),
		TokenIDs:  []string{"test"},
	}

	mock.ExpectHSet("session-12345", adapter.serializeStruct(mySession)...).SetVal(1)

	err := adapter.SetSession(ctx, mySession)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetSession(t *testing.T) {
	ctx := context.Background()

	client, mock := redismock.NewClientMock()

	adapter := RedisAdapter{
		rdb: client,
	}

	session := models.Session{
		ID:        "12345",
		ExpiresAt: time.Now().Add(time.Second * time.Duration(uint64(rand.Int63n(14400)))),
		TokenIDs:  models.SerializableStringSlice{"1", "2"},
	}
	sessionMap, err := decomposeStructToMap(session)
	assert.NoError(t, err)
	mock.ExpectHGetAll(sessionPrefix + session.ID).SetVal(sessionMap)

	outputSession, err := adapter.GetSession(ctx, session.ID)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
	assert.Truef(
		t,
		cmp.Equal(session, outputSession),
		fmt.Sprintf("The two values are not equal, diff is: %s\n", cmp.Diff(session, outputSession)),
	)
}

func TestRemoveSession(t *testing.T) {
	ctx := context.Background()

	client, mock := redismock.NewClientMock()

	adapter := RedisAdapter{
		rdb: client,
	}

	mock.ExpectDel(sessionPrefix + "12345").SetVal(1)

	err := adapter.RemoveSession(ctx, "12345")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSetAccessToken(t *testing.T) {
	ctx := context.Background()

	client, mock := redismock.NewClientMock()

	adapter := RedisAdapter{
		rdb: client,
	}

	myAccessToken := models.OauthToken{
		ID:        "12345",
		Value:     "6789",
		ExpiresAt: time.Now().Add(time.Second * time.Duration(uint64(rand.Int63n(14400)))),
		TokenURL:  "https://gitlab.com",
		Type:      models.AccessTokenType,
	}

	z1 := redis.Z{
		Score:  float64(myAccessToken.ExpiresAt.Unix()),
		Member: myAccessToken.ID,
	}

	mock.ExpectZAdd(indexExpiringTokens, z1).SetVal(1)
	mock.ExpectHSet(
		accessTokenPrefix+"12345",
		adapter.serializeStruct(myAccessToken)...,
	).SetVal(1)

	err := adapter.SetAccessToken(ctx, myAccessToken)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAccessToken(t *testing.T) {
	ctx := context.Background()

	client, mock := redismock.NewClientMock()

	adapter := RedisAdapter{
		rdb: client,
	}

	token := models.OauthToken{
		ID:        "12345",
		Value:     "6789",
		ExpiresAt: time.Now().Add(time.Second * time.Duration(uint64(rand.Int63n(14400)))),
		TokenURL:  "https://gitlab.com",
		Type:      models.AccessTokenType,
	}
	tokenMap, err := decomposeStructToMap(token)
	assert.NoError(t, err)

	mock.ExpectHGetAll(accessTokenPrefix + "12345").SetVal(tokenMap)
	outputToken, err := adapter.GetAccessToken(ctx, "12345")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
	assert.Truef(
		t,
		cmp.Equal(token, outputToken, compareOptions...),
		"The two values are not equal, diff is: %s\n",
		cmp.Diff(token, outputToken, compareOptions...),
	)
}

func TestRemoveAccessToken(t *testing.T) {
	ctx := context.Background()

	client, mock := redismock.NewClientMock()

	adapter := RedisAdapter{
		rdb: client,
	}

	myAccessToken := models.OauthToken{
		ID:        "12345",
		Value:     "6789",
		ExpiresAt: time.Now().Add(time.Second * time.Duration(uint64(rand.Int63n(14400)))),
		TokenURL:  "https://gitlab.com",
		Type:      models.AccessTokenType,
	}

	z1 := redis.Z{
		Score:  float64(myAccessToken.ExpiresAt.Unix()),
		Member: myAccessToken.ID,
	}

	mock.ExpectZRem("indexExpiringTokens", z1).SetVal(1)
	mock.ExpectDel(accessTokenPrefix + "12345").SetVal(1)

	err := adapter.RemoveAccessToken(ctx, myAccessToken)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSetRefreshToken(t *testing.T) {
	ctx := context.Background()

	client, mock := redismock.NewClientMock()

	adapter := RedisAdapter{
		rdb: client,
	}

	token := models.OauthToken{
		ID:        "12345",
		Value:     "6789",
		ExpiresAt: time.Now().Add(time.Second * time.Duration(uint64(rand.Int63n(14400)))),
		Type:      models.RefreshTokenType,
	}

	mock.ExpectHSet(refreshTokenPrefix+"12345", adapter.serializeStruct(token)...).SetVal(1)

	err := adapter.SetRefreshToken(ctx, token)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetRefreshToken(t *testing.T) {
	ctx := context.Background()

	client, mock := redismock.NewClientMock()

	adapter := RedisAdapter{
		rdb: client,
	}

	token := models.OauthToken{
		ID:        "12345",
		Value:     "6789",
		ExpiresAt: time.Now().Add(time.Second * time.Duration(uint64(rand.Int63n(14400)))),
		Type:      models.RefreshTokenType,
	}
	tokenMap, err := decomposeStructToMap(token)
	assert.NoError(t, err)

	mock.ExpectHGetAll(refreshTokenPrefix + "12345").SetVal(tokenMap)

	outputToken, err := adapter.GetRefreshToken(ctx, "12345")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
	assert.Truef(
		t,
		cmp.Equal(token, outputToken, compareOptions...),
		"The two values are not equal, diff is: %s\n",
		cmp.Diff(token, outputToken, compareOptions...),
	)
}

func TestRemoveRefreshToken(t *testing.T) {
	ctx := context.Background()

	client, mock := redismock.NewClientMock()

	adapter := RedisAdapter{
		rdb: client,
	}

	mock.ExpectDel(refreshTokenPrefix + "12345").SetVal(1)

	err := adapter.RemoveRefreshToken(ctx, "12345")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetExpiringAccessTokenIDs(t *testing.T) {
	ctx := context.Background()

	client, mock := redismock.NewClientMock()

	adapter := RedisAdapter{
		rdb: client,
	}

	startTime := time.Now()

	stopTime := time.Now().Add(time.Hour * 4)

	zRangeArgs := redis.ZRangeArgs{
		Key:     indexExpiringTokens,
		Start:   startTime.Unix(),
		Stop:    stopTime.Unix(),
		ByScore: true,
	}
	zVals := []redis.Z{
		{Score: float64(startTime.Add(time.Hour).Unix()), Member: "id1"},
		{Score: float64(startTime.Add(time.Hour * 2).Unix()), Member: "id2"},
	}
	mock.ExpectZRangeArgsWithScores(zRangeArgs).SetVal(zVals)
	output, err := adapter.GetExpiringAccessTokenIDs(ctx, startTime, stopTime)
	assert.NoError(t, err)
	assert.Equal(t, []string{"id1", "id2"}, output)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSetProjectToken(t *testing.T) {
	ctx := context.Background()

	client, mock := redismock.NewClientMock()

	adapter := RedisAdapter{
		rdb: client,
	}

	token := models.OauthToken{
		ID:        "12345",
		Value:     "6789",
		ExpiresAt: time.Now().Add(time.Second * time.Duration(uint64(rand.Int63n(14400)))),
		TokenURL:  "https://gitlab.com",
		Type:      models.AccessTokenType,
	}

	z1 := redis.Z{
		Score:  float64(token.ExpiresAt.Unix()),
		Member: token.ID,
	}

	mock.ExpectZAdd(projectTokenPrefix+"4567", z1).SetVal(1)
	err := adapter.SetProjectToken(ctx, 4567, token)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetProjectTokens(t *testing.T) {
	ctx := context.Background()

	client, mock := redismock.NewClientMock()

	adapter := RedisAdapter{
		rdb: client,
	}

	zRangeArgs := redis.ZRangeArgs{
		Key:     projectTokenPrefix + "4567",
		Start:   0,
		Stop:    999999,
		ByScore: false,
	}
	zVals := []redis.Z{
		{Score: 10, Member: "id1"},
		{Score: 20, Member: "id2"},
	}
	mock.ExpectZRangeArgsWithScores(zRangeArgs).SetVal(zVals)

	ids, err := adapter.GetProjectTokens(ctx, 4567)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
	assert.Equal(t, []string{"id1", "id2"}, ids)
}

func TestRemoveProjectToken(t *testing.T) {
	ctx := context.Background()

	client, mock := redismock.NewClientMock()

	adapter := RedisAdapter{
		rdb: client,
	}

	myAccessToken := models.OauthToken{
		ID:        "12345",
		Value:     "6789",
		ExpiresAt: time.Now().Add(time.Second * time.Duration(uint64(rand.Int63n(14400)))),
		TokenURL:  "https://gitlab.com",
		Type:      models.AccessTokenType,
	}

	z1 := redis.Z{
		Score:  float64(myAccessToken.ExpiresAt.Unix()),
		Member: myAccessToken.ID,
	}

	mock.ExpectZRem(projectTokenPrefix+"4567", z1).SetVal(1)

	err := adapter.RemoveProjectToken(ctx, 4567, myAccessToken)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

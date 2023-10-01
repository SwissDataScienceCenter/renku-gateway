package db

import (
	"context"
	"encoding"
	"fmt"
	"math/rand"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
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

func TestSetGetSession(t *testing.T) {
	ctx := context.Background()
	adapter := NewMockRedisAdapter()
	mySession := models.Session{
		ID:        "12345",
		ExpiresAt: time.Now().Add(time.Second * time.Duration(uint64(rand.Int63n(14400)))),
		TokenIDs:  []string{"test"},
	}
	err := adapter.SetSession(ctx, mySession)
	assert.NoError(t, err)
	session, err := adapter.GetSession(ctx, "12345")
	assert.NoError(t, err)
	assert.Truef(
		t,
		cmp.Equal(mySession, session, compareOptions...),
		"The two values are not equal, diff is: %s\n",
		cmp.Diff(mySession, session, compareOptions...),
	)
}

func TestRemoveSession(t *testing.T) {
	ctx := context.Background()
	adapter := NewMockRedisAdapter()
	mySession := models.Session{
		ID:        "12345",
		ExpiresAt: time.Now().Add(time.Second * time.Duration(uint64(rand.Int63n(14400)))),
		TokenIDs:  []string{"test"},
	}
	err := adapter.SetSession(ctx, mySession)
	assert.NoError(t, err)
	err = adapter.RemoveSession(ctx, "12345")
	assert.NoError(t, err)
	session, err := adapter.GetSession(ctx, "12345")
	assert.NoError(t, err)
	assert.Equal(t, models.Session{}, session)
}

func TestSetGetRemoveAccessToken(t *testing.T) {
	ctx := context.Background()
	adapter := NewMockRedisAdapter()
	myAccessToken := models.OauthToken{
		ID:        "12345",
		Value:     "6789",
		ExpiresAt: time.Now().Add(time.Hour * 24),
		TokenURL:  "https://gitlab.com",
		Type:      models.AccessTokenType,
	}
	err := adapter.SetAccessToken(ctx, myAccessToken)
	assert.NoError(t, err)
	accessToken, err := adapter.GetAccessToken(ctx, myAccessToken.ID)
	assert.NoError(t, err)
	assert.Truef(
		t,
		cmp.Equal(myAccessToken, accessToken, compareOptions...),
		"The two values are not equal, diff is: %s\n",
		cmp.Diff(myAccessToken, accessToken, compareOptions...),
	)
	ids, err := adapter.GetExpiringAccessTokenIDs(ctx, time.Now(), time.Now().Add(time.Hour))
	assert.NoError(t, err)
	assert.Len(t, ids, 0)
	ids, err = adapter.GetExpiringAccessTokenIDs(ctx, time.Now(), time.Now().Add(time.Hour*999))
	assert.NoError(t, err)
	assert.Len(t, ids, 1)
	err = adapter.RemoveAccessToken(ctx, myAccessToken)
	assert.NoError(t, err)
	accessToken, err = adapter.GetAccessToken(ctx, myAccessToken.ID)
	assert.NoError(t, err)
	assert.Equal(t, models.OauthToken{}, accessToken)
}

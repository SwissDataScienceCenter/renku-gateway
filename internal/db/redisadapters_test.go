package db

import (
	"context"
	"crypto/rand"
	"encoding"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/gwerrors"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var compareOptions []cmp.Option = []cmp.Option{cmpopts.IgnoreUnexported(models.OauthToken{})}

func decomposeStructToMap(strct any) (map[string]string, error) {
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
	mySession, err := models.NewSession()
	require.NoError(t, err)
	err = adapter.SetSession(ctx, mySession)
	require.NoError(t, err)
	session, err := adapter.GetSession(ctx, mySession.ID)
	require.NoError(t, err)
	assert.True(t, mySession.Equal(&session))
}

func TestRemoveSession(t *testing.T) {
	ctx := context.Background()
	adapter := NewMockRedisAdapter()
	mySession, err := models.NewSession()
	require.NoError(t, err)
	err = adapter.SetSession(ctx, mySession)
	assert.NoError(t, err)
	err = adapter.RemoveSession(ctx, mySession.ID)
	assert.NoError(t, err)
	_, err = adapter.GetSession(ctx, mySession.ID)
	assert.ErrorIs(t, err, gwerrors.ErrSessionNotFound)
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
	assert.ErrorIs(t, err, gwerrors.ErrTokenNotFound)
}

func TestSetGetAccessTokenWithEncryption(t *testing.T) {
	ctx := context.Background()
	secretKey := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, secretKey)
	require.NoError(t, err)
	adapter := NewMockRedisAdapter(WithEncryption(string(secretKey)))
	myAccessToken := models.OauthToken{
		ID:        "12345",
		Value:     "6789",
		ExpiresAt: time.Now().Add(time.Hour * 24),
		TokenURL:  "https://gitlab.com",
		Type:      models.AccessTokenType,
	}
	err = adapter.SetAccessToken(ctx, myAccessToken)
	assert.NoError(t, err)
	accessToken, err := adapter.GetAccessToken(ctx, myAccessToken.ID)
	assert.NoError(t, err)
	assert.Truef(
		t,
		cmp.Equal(myAccessToken, accessToken, compareOptions...),
		"The two values are not equal, diff is: %s\n",
		cmp.Diff(myAccessToken, accessToken, compareOptions...),
	)
}

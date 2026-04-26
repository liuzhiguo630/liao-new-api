package controller

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestShouldRetry_AwsInvokeBadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	err := types.NewOpenAIError(
		errors.New("InvokeModel: ValidationException: Access to Bedrock models is not allowed for this account"),
		types.ErrorCodeAwsInvokeError,
		http.StatusBadRequest,
	)

	require.True(t, shouldRetry(c, err, 1))
}

func TestShouldRetry_NormalBadRequestStillNoRetry(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	err := types.NewOpenAIError(
		errors.New("bad request"),
		types.ErrorCodeInvalidRequest,
		http.StatusBadRequest,
	)

	require.False(t, shouldRetry(c, err, 1))
}

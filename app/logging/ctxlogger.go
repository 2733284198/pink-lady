package logging

import (
	"github.com/gin-gonic/gin"
	uuid "github.com/satori/go.uuid"
	"go.uber.org/zap"
)

const (
	// CtxLoggerKey define the logger keyname which in context
	CtxLoggerKey = "ctxLogger"
	// RequestIDKey define the request id header key
	RequestIDKey = "X-Request-Id"
)

// SetCtxLogger add a logger with given field into the gin.Context
// and set requestid field get from context
func SetCtxLogger(c *gin.Context, ctxLogger *zap.Logger) {
	c.Set(CtxLoggerKey, ctxLogger)
}

// CtxLogger get the ctxLogger in gin.Context
func CtxLogger(c *gin.Context, fields ...zap.Field) *zap.Logger {
	var ctxLogger *zap.Logger
	ctxLoggerItf, _ := c.Get(CtxLoggerKey)
	if ctxLoggerItf != nil {
		ctxLogger = ctxLoggerItf.(*zap.Logger)
	} else {
		ctxLogger = CloneLogger().With(zap.String("requestid", CtxRequestID(c)))
	}
	if len(fields) > 0 {
		ctxLogger = ctxLogger.With(fields...)
	}
	return ctxLogger
}

// CtxRequestID get requestid from context
func CtxRequestID(c *gin.Context) string {
	// first get from context
	requestidItf, _ := c.Get(RequestIDKey)
	if requestidItf != nil {
		requestid := requestidItf.(string)
		if requestid != "" {
			return requestid
		}
	}
	// if not then get request id from header
	requestid := c.Request.Header.Get(RequestIDKey)
	if requestid != "" {
		return requestid
	}
	// else gen a request id
	return uuid.NewV4().String()
}

// SetCtxRequestID set requestid for context
func SetCtxRequestID(c *gin.Context, requestid string) {
	// set in context
	c.Set(RequestIDKey, requestid)
	// set in header
	c.Writer.Header().Set(RequestIDKey, requestid)
}
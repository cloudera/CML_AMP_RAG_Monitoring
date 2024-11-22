package sbhttptest

import (
	"bytes"
	"encoding/json"
	lhttptest "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/http/test"
	sbhttpbase "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/serverbase/http/base"
	"go.uber.org/zap"
	"io"
	"net/http"
	"net/http/httptest"
	"pgregory.net/rapid"
)

func JsonObjectGenerator(maxJsonDepth int) *rapid.Generator[json.RawMessage] {
	/*
		Excerpt from https://www.w3schools.com/js/js_json_datatypes.asp:
			In JSON, values must be one of the following data types:
			a string, a number, an object (JSON object), an array, a boolean, null
	*/
	// TODO: handle more generic json
	return rapid.Custom(func(t *rapid.T) json.RawMessage {
		generators := []*rapid.Generator[json.RawMessage]{
			jsonBoolGenerator(),
			jsonNullGenerator(),
			jsonNumberGenerator(),
			jsonStringGenerator(),
		}
		if maxJsonDepth > 0 {
			generators = append(generators,
				jsonArrayGenerator(maxJsonDepth-1),
				jsonMapGenerator(maxJsonDepth-1),
			)
		}
		return rapid.OneOf(generators...).Draw(t, "json")
	})
}

func jsonArrayGenerator(maxJsonDepth int) *rapid.Generator[json.RawMessage] {
	return rapid.Custom(func(t *rapid.T) json.RawMessage {
		ret, _ := json.Marshal(rapid.SliceOfN(JsonObjectGenerator(maxJsonDepth), 1, 5).Draw(t, "array"))
		return ret
	})
}

func jsonBoolGenerator() *rapid.Generator[json.RawMessage] {
	return rapid.Custom(func(t *rapid.T) json.RawMessage {
		ret, _ := json.Marshal(rapid.Bool().Draw(t, "bool"))
		return ret
	})
}

func jsonMapGenerator(maxJsonDepth int) *rapid.Generator[json.RawMessage] {
	return rapid.Custom(func(t *rapid.T) json.RawMessage {
		ret, _ := json.Marshal(rapid.MapOfN(rapid.StringMatching(`[a-z]{1,10}`), JsonObjectGenerator(maxJsonDepth), 1, 5).Draw(t, "map"))
		return ret
	})
}

func jsonNullGenerator() *rapid.Generator[json.RawMessage] {
	return rapid.Just(json.RawMessage("null"))
}

func jsonNumberGenerator() *rapid.Generator[json.RawMessage] {
	return rapid.Custom(func(t *rapid.T) json.RawMessage {
		ret, _ := json.Marshal(rapid.Float64().Draw(t, "float"))
		return ret
	})
}

func jsonStringGenerator() *rapid.Generator[json.RawMessage] {
	return rapid.Custom(func(t *rapid.T) json.RawMessage {
		ret, _ := json.Marshal(rapid.StringMatching(`[a-z]{1,10}`).Draw(t, "string"))
		return ret
	})
}

func RequestGenerator(recorder *httptest.ResponseRecorder) *rapid.Generator[*sbhttpbase.Request] {
	return rapid.Custom(func(t *rapid.T) *sbhttpbase.Request {
		body := rapid.SliceOf(rapid.Byte()).Draw(t, "body")
		bodyBuffer := bytes.NewBuffer(body)
		return requestGeneratorHelper(t, recorder, bodyBuffer)
	})
}

func RequestWithBodyGenerator(recorder *httptest.ResponseRecorder, body io.Reader) *rapid.Generator[*sbhttpbase.Request] {
	return rapid.Custom(func(t *rapid.T) *sbhttpbase.Request {
		return requestGeneratorHelper(t, recorder, body)
	})
}

func requestGeneratorHelper(t *rapid.T, recorder *httptest.ResponseRecorder, body io.Reader) *sbhttpbase.Request {
	params := rapid.MapOf(rapid.String(), rapid.String()).Draw(t, "params")

	request := &sbhttpbase.Request{
		PathPattern: rapid.String().Draw(t, "path"),
		Logger:      zap.NewNop(),
		Writer:      recorder,
		Request: httptest.NewRequest(
			lhttptest.MethodGenerator().Draw(t, "method"),
			lhttptest.UrlGenerator().Draw(t, "target"),
			body,
		),
		Params: params,
	}

	headers := lhttptest.HeadersGenerator().Draw(t, "header")
	request.Request.Header = headers
	return request
}

func OkHandlerGenerator() *rapid.Generator[sbhttpbase.HandleFunc] {
	return rapid.Custom(func(t *rapid.T) sbhttpbase.HandleFunc {
		headers := lhttptest.HeadersGenerator().Draw(t, "headers")
		body := rapid.SliceOf(rapid.Byte()).Draw(t, "body")

		return func(request *sbhttpbase.Request) {
			// TODO: capture error?
			io.Copy(io.Discard, request.Request.Body)
			request.Request.Body.Close()

			writer := request.Writer

			for k, vals := range headers {
				writer.Header().Del(k)
				for _, v := range vals {
					writer.Header().Add(k, v)
				}
			}

			writer.WriteHeader(http.StatusOK)
			// TODO: capture error?
			writer.Write(body)
		}
	})
}

func HandlerGenerator() *rapid.Generator[sbhttpbase.HandleFunc] {
	return rapid.Custom(func(t *rapid.T) sbhttpbase.HandleFunc {
		code := lhttptest.CodeGenerator().Draw(t, "code")
		headers := lhttptest.HeadersGenerator().Draw(t, "headers")
		body := rapid.SliceOf(rapid.Byte()).Draw(t, "body")

		return func(request *sbhttpbase.Request) {
			// TODO: capture error?
			io.Copy(io.Discard, request.Request.Body)
			request.Request.Body.Close()

			writer := request.Writer

			for k, vals := range headers {
				writer.Header().Del(k)
				for _, v := range vals {
					writer.Header().Add(k, v)
				}
			}

			writer.WriteHeader(code)
			// TODO: capture error?
			writer.Write(body)
		}
	})
}

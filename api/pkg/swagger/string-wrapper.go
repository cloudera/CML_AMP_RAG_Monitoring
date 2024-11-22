package swagger

import (
	"bytes"
	"encoding/json"
	"io"
)

type StringWrapper struct {
	bytes.Buffer
	content string
}

func (s *StringWrapper) GetContent() (string, error) {
	if s.content == "" {
		s.content = s.String()
	}

	return s.content, nil
}

func (s *StringWrapper) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &s.content)
}

var _ io.Writer = &StringWrapper{}

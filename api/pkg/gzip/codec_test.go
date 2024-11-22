package lgzip

import (
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
	"pgregory.net/rapid"
	"testing"
)

type testTransformer func(*rapid.T, []byte) []byte
type testWriter func(writer io.Writer) (io.WriteCloser, error)
type testReader func(reader io.Reader) (io.ReadCloser, error)

func testTransformerNop(_ *rapid.T, b []byte) []byte { return b }
func testTransformCompress(t *rapid.T, b []byte) []byte {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	_, err := w.Write(b)
	if err != nil {
		t.Fatalf("failed to compress: %s", err.Error())
	}
	err = w.Close()
	if err != nil {
		t.Fatalf("failed to close: %s", err.Error())
	}
	return buf.Bytes()
}
func testTransformDecompress(t *rapid.T, b []byte) []byte {
	buf := bytes.NewBuffer(b)
	r, err := gzip.NewReader(buf)
	if err != nil {
		t.Fatalf("failed to read: %s", err.Error())
	}
	data, err := ioutil.ReadAll(r)
	if err != nil {
		t.Fatalf("failed to read: %s", err.Error())
	}
	return data
}

func TestCodec(t *testing.T) {
	tests := []struct {
		pre    testTransformer
		post   testTransformer
		writer testWriter
		reader testReader
	}{
		{testTransformerNop, testTransformerNop, NewDecompressWriter, NewCompressReader},
		{testTransformCompress, testTransformDecompress, func(writer io.Writer) (io.WriteCloser, error) { return NewCompressWriter(writer) }, func(reader io.Reader) (io.ReadCloser, error) { return NewDecompressReader(reader) }},
	}

	for _, tc := range tests {
		rapid.Check(t, func(t *rapid.T) {
			data := rapid.SliceOf(rapid.Byte()).Draw(t, "data")
			result := &bytes.Buffer{}

			writer, err := NewDecompressWriter(result)
			if err != nil {
				t.Fatal(err)
			}

			reader, err := NewCompressReader(bytes.NewBuffer(tc.pre(t, data)))
			if err != nil {
				t.Fatal(err)
			}

			if _, err := io.Copy(writer, reader); err != nil {
				t.Fatal(err)
			}

			if err := reader.Close(); err != nil {
				t.Fatal(err)
			}

			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}

			if bytes.Compare(data, tc.post(t, result.Bytes())) != 0 {
				t.Fatalf("data doesn't match")
			}
		})
	}
}

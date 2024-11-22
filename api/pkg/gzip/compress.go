package lgzip

import (
	"compress/gzip"
	"io"
	"sync"
)

func NewCompressReader(r io.Reader) (io.ReadCloser, error) {
	return NewCompressReaderLevel(r, gzip.DefaultCompression)
}

func NewCompressWriter(w io.Writer) (*gzip.Writer, error) {
	return NewCompressWriterLevel(w, gzip.DefaultCompression)
}

func NewCompressReaderLevel(r io.Reader, level int) (io.ReadCloser, error) {
	pr, pw := io.Pipe()

	newReader := compressReader{
		pr: pr,
	}
	newReader.wg.Add(1)

	go func() {
		defer newReader.wg.Done()

		gzipWriter, err := gzip.NewWriterLevel(pw, level)
		if err != nil {
			pw.CloseWithError(err)
			return
		}

		if _, err := io.Copy(gzipWriter, r); err != nil && err != io.EOF {
			pw.CloseWithError(err)
			return
		}

		if err := gzipWriter.Close(); err != nil {
			pw.CloseWithError(err)
			return
		}

		pw.Close()
	}()

	return &newReader, nil
}

func NewCompressWriterLevel(w io.Writer, level int) (*gzip.Writer, error) {
	return gzip.NewWriterLevel(w, level)
}

type compressReader struct {
	pr *io.PipeReader
	wg sync.WaitGroup
}

func (r *compressReader) Read(p []byte) (n int, err error) {
	return r.pr.Read(p)
}

func (r *compressReader) Close() error {
	if err := r.pr.Close(); err != nil {
		return err
	}
	r.wg.Wait()
	return nil
}

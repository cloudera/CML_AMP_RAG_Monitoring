package lgzip

import (
	"compress/gzip"
	"io"
	"sync"
)

func NewDecompressReader(r io.Reader) (*gzip.Reader, error) {
	return gzip.NewReader(r)
}

func NewDecompressWriter(w io.Writer) (io.WriteCloser, error) {
	pr, pw := io.Pipe()

	newWriter := decompressWriter{
		pw: pw,
	}
	newWriter.wg.Add(1)

	go func() {
		defer newWriter.wg.Done()

		gzipReader, err := gzip.NewReader(pr)
		if err != nil {
			pr.CloseWithError(err)
			return
		}

		if _, err := io.Copy(w, gzipReader); err != nil && err != io.EOF {
			pr.CloseWithError(err)
			return
		}

		if err := gzipReader.Close(); err != nil {
			pr.CloseWithError(err)
			return
		}

		pr.Close()
	}()

	return &newWriter, nil
}

type decompressWriter struct {
	pw *io.PipeWriter
	wg sync.WaitGroup
}

func (w *decompressWriter) Write(p []byte) (n int, err error) {
	return w.pw.Write(p)
}

func (w *decompressWriter) Close() error {
	if err := w.pw.Close(); err != nil {
		return err
	}
	w.wg.Wait()
	return nil
}

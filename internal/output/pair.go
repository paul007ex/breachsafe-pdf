// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

// Package output persists a PDF and its adjacent result without clobbering.
package output

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/paul007ex/breachsafe-pdf/internal/fault"
)

// Pair contains the exact two output byte sequences.
type Pair struct {
	PDFPath     string
	PDFBytes    []byte
	ResultPath  string
	ResultBytes []byte
}

// WritePair stages both files and commits them with no-clobber hard links.
// Cross-file atomicity is unavailable on normal filesystems. If committing the
// result fails, this function removes the PDF it created before returning.
func WritePair(ctx context.Context, pair Pair) error {
	if err := ctx.Err(); err != nil {
		return fault.Wrap(fault.CodeCanceled, "output.write_pair", err)
	}
	if pair.PDFPath == "" || pair.ResultPath == "" {
		return fault.New(fault.CodeInvalidInput, "output.write_pair", "path", "both output paths are required")
	}
	pdfAbs, err := filepath.Abs(pair.PDFPath)
	if err != nil {
		return fault.WrapField(fault.CodeInvalidInput, "output.path", "pdf", err)
	}
	resultAbs, err := filepath.Abs(pair.ResultPath)
	if err != nil {
		return fault.WrapField(fault.CodeInvalidInput, "output.path", "result", err)
	}
	if pdfAbs == resultAbs {
		return fault.New(fault.CodeInvalidInput, "output.write_pair", "path", "PDF and result paths must differ")
	}
	if len(pair.PDFBytes) == 0 || len(pair.ResultBytes) == 0 {
		return fault.New(fault.CodeInvalidInput, "output.write_pair", "bytes", "both outputs must be non-empty")
	}

	pdfStage, err := stage(pdfAbs, pair.PDFBytes)
	if err != nil {
		return err
	}
	defer removeStage(pdfStage)
	resultStage, err := stage(resultAbs, pair.ResultBytes)
	if err != nil {
		return err
	}
	defer removeStage(resultStage)

	if err := ctx.Err(); err != nil {
		return fault.Wrap(fault.CodeCanceled, "output.write_pair", err)
	}
	if err := commit(pdfStage, pdfAbs); err != nil {
		return err
	}
	if err := commit(resultStage, resultAbs); err != nil {
		cleanupErr := os.Remove(pdfAbs)
		if cleanupErr != nil {
			return fault.New(fault.CodeWriteFailed, "output.rollback", "pdf", fmt.Sprintf("result commit failed (%v); PDF cleanup failed (%v)", err, cleanupErr))
		}
		return err
	}
	return nil
}

func stage(destination string, data []byte) (string, error) {
	directory := filepath.Dir(destination)
	file, err := os.CreateTemp(directory, ".breachsafe-pdf-stage-*")
	if err != nil {
		return "", fault.Wrap(fault.CodeWriteFailed, "output.stage", err)
	}
	path := file.Name()
	ok := false
	defer func() {
		if !ok {
			if closeErr := file.Close(); closeErr != nil {
				removeStage(path)
				return
			}
			removeStage(path)
		}
	}()
	if err := file.Chmod(0o600); err != nil {
		return "", fault.Wrap(fault.CodeWriteFailed, "output.stage_mode", err)
	}
	if _, err := file.Write(data); err != nil {
		return "", fault.Wrap(fault.CodeWriteFailed, "output.stage_write", err)
	}
	if err := file.Sync(); err != nil {
		return "", fault.Wrap(fault.CodeWriteFailed, "output.stage_sync", err)
	}
	if err := file.Close(); err != nil {
		return "", fault.Wrap(fault.CodeWriteFailed, "output.stage_close", err)
	}
	ok = true
	return path, nil
}

func removeStage(path string) {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		// Staging cleanup is best effort after the primary operation has failed.
		return
	}
}

func commit(staged, destination string) error {
	if err := os.Link(staged, destination); err != nil {
		if errors.Is(err, os.ErrExist) {
			return fault.New(fault.CodeOutputExists, "output.commit", filepath.Base(destination), "refusing to overwrite existing output")
		}
		return fault.Wrap(fault.CodeWriteFailed, "output.commit", err)
	}
	return nil
}

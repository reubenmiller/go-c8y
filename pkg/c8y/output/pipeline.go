// Package output provides a lazy, streaming pipeline for processing
// Cumulocity API responses: filtering, shaping and rendering collection
// items (JSON, NDJSON, CSV, TSV, ...) without buffering whole collections.
//
// The pipeline is pull-based, built on iter.Seq2: no work happens until the
// renderer asks for the next item, and stopping consumption stops the source
// (including pagination when the source is a pagination.Iterator). This gives
// lazy execution and backpressure without channels or goroutines.
//
// Stages are plain functions over the item stream, so custom logic is just a
// function — see Map, Filter and Head for the core building blocks, and the
// filter and shape subpackages for compiled predicates and property selection.
package output

import (
	"context"
	"iter"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
)

// Seq is the stream type that flows through the pipeline: each element is a
// single JSON document (one item of a collection) paired with any error
// encountered while producing it.
type Seq = iter.Seq2[jsondoc.JSONDoc, error]

// Stage transforms the item stream. Stages must be lazy: per-item work may
// only happen inside the returned sequence, and expensive setup (compiling
// patterns, templates, ...) should happen when the Stage is constructed.
type Stage func(Seq) Seq

// Renderer consumes documents and writes formatted output. Close must be
// called once after the last Write to flush buffered output and write any
// trailing content.
type Renderer interface {
	Write(doc jsondoc.JSONDoc) error
	Close() error
}

// Compose combines stages into a single stage, applied in argument order.
// Nil stages are skipped.
func Compose(stages ...Stage) Stage {
	return func(src Seq) Seq {
		for _, s := range stages {
			if s != nil {
				src = s(src)
			}
		}
		return src
	}
}

// Render pulls items from src through the given stages and writes them with r.
// It returns the first error encountered (source, stage, renderer or context
// cancellation). On success the renderer is closed; on error it is left
// unclosed so trailing content is not written after partial output.
func Render(ctx context.Context, src Seq, r Renderer, stages ...Stage) error {
	for doc, err := range Compose(stages...)(src) {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := r.Write(doc); err != nil {
			return err
		}
	}
	return r.Close()
}

// Map returns a stage that transforms each document with fn. An error from fn
// is yielded downstream (stopping Render with that error).
func Map(fn func(jsondoc.JSONDoc) (jsondoc.JSONDoc, error)) Stage {
	return func(src Seq) Seq {
		return func(yield func(jsondoc.JSONDoc, error) bool) {
			mapSeq(src, fn, yield)
		}
	}
}

func mapSeq(src Seq, fn func(jsondoc.JSONDoc) (jsondoc.JSONDoc, error), yield func(jsondoc.JSONDoc, error) bool) {
	for doc, err := range src {
		if err != nil {
			if !yield(doc, err) {
				return
			}
			continue
		}
		out, err := fn(doc)
		if !yield(out, err) {
			return
		}
	}
}

// Filter returns a stage that keeps only documents matching pred.
// Errors from upstream are passed through.
func Filter(pred func(jsondoc.JSONDoc) bool) Stage {
	return func(src Seq) Seq {
		return func(yield func(jsondoc.JSONDoc, error) bool) {
			filterSeq(src, pred, yield)
		}
	}
}

func filterSeq(src Seq, pred func(jsondoc.JSONDoc) bool, yield func(jsondoc.JSONDoc, error) bool) {
	for doc, err := range src {
		if err != nil {
			if !yield(doc, err) {
				return
			}
			continue
		}
		if !pred(doc) {
			continue
		}
		if !yield(doc, nil) {
			return
		}
	}
}

// Head returns a stage that ends the stream after n documents. Because the
// pipeline is pull-based, this also stops the source, e.g. no further pages
// are fetched from a paginated source.
func Head(n int) Stage {
	return func(src Seq) Seq {
		return func(yield func(jsondoc.JSONDoc, error) bool) {
			headSeq(src, n, yield)
		}
	}
}

func headSeq(src Seq, n int, yield func(jsondoc.JSONDoc, error) bool) {
	if n <= 0 {
		return
	}
	count := 0
	for doc, err := range src {
		if !yield(doc, err) {
			return
		}
		if err == nil {
			count++
			if count >= n {
				return
			}
		}
	}
}

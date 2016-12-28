//
// Copyright 2016 Gregory Trubetskoy. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dsl

import (
	"time"

	"github.com/tgres/tgres/rrd"
	"github.com/tgres/tgres/serde"
	"github.com/tgres/tgres/series"
)

// This is a subset of serde.Fetcher
type dsFetcher interface {
	serde.DataSourceNamesFetcher
	FetchDataSourceById(id int64) (rrd.DataSourcer, error)
	FetchSeries(ds rrd.DataSourcer, from, to time.Time, maxPoints int64) (series.Series, error)
}

type fsFinder interface {
	dsIdsFromIdent(ident string) map[string]int64
	FsFind(pattern string) []*FsFindNode
}

type ReadCacher interface {
	dsFetcher
	fsFinder
}

type readCache struct {
	dsFetcher
	dsns *dataSourceNames
}

func NewReadCache(db dsFetcher) *readCache {
	return &readCache{dsFetcher: db, dsns: &dataSourceNames{}}
}

func (r *readCache) dsIdsFromIdent(ident string) map[string]int64 {
	result := r.dsns.dsIdsFromIdent(ident)
	if len(result) == 0 {
		r.dsns.reload(r)
		result = r.dsns.dsIdsFromIdent(ident)
	}
	return result
}

// FsFind provides a way of searching dot-separated names using same
// rules as filepath.Match, as well as comma-separated values in curly
// braces such as "foo.{bar,baz}".
func (r *readCache) FsFind(pattern string) []*FsFindNode {
	r.dsns.reload(r)
	return r.dsns.fsFind(pattern)
}

// Creates a readCache from a map of DataSourcers.
func NewReadCacheFromMap(dss map[string]rrd.DataSourcer) *readCache {
	return NewReadCache(newMapCache(dss))
}

// A dsFinder backed by a simple map of DSs
func newMapCache(dss map[string]rrd.DataSourcer) *mapCache {
	mc := &mapCache{make(map[string]int64), make(map[int64]rrd.DataSourcer)}
	var n int64
	for name, ds := range dss {
		mc.byName[name] = n
		mc.byId[n] = ds
		n++
	}
	return mc
}

type mapCache struct {
	byName map[string]int64
	byId   map[int64]rrd.DataSourcer
}

func (m *mapCache) FetchDataSourceNames() (map[string]int64, error) {
	return m.byName, nil
}

func (m *mapCache) FetchDataSourceById(id int64) (rrd.DataSourcer, error) {
	return m.byId[id], nil
}

func (*mapCache) FetchSeries(ds rrd.DataSourcer, from, to time.Time, maxPoints int64) (series.Series, error) {
	return series.NewRRASeries(ds.RRAs()[0]), nil
}

func (m *mapCache) FetchDataSources() ([]rrd.DataSourcer, error) {
	result := []rrd.DataSourcer{}
	for _, ds := range m.byId {
		result = append(result, ds)
	}
	return result, nil
}

func (*mapCache) FetchOrCreateDataSource(name string, dsSpec *rrd.DSSpec) (rrd.DataSourcer, error) {
	return nil, nil
}

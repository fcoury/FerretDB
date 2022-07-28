// Copyright 2021 FerretDB Inc.
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

package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/FerretDB/FerretDB/integration/setup"
	"github.com/FerretDB/FerretDB/integration/shareddata"
)

// This file is for all remaining update tests.

func TestUpdateUpsert(t *testing.T) {
	t.Parallel()
	ctx, collection := setup.Setup(t, shareddata.Composites)

	// this upsert inserts document
	filter := bson.D{{"foo", "bar"}}
	update := bson.D{{"$set", bson.D{{"foo", "baz"}}}}
	res, err := collection.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	require.NoError(t, err)

	id := res.UpsertedID
	assert.NotEmpty(t, id)
	res.UpsertedID = nil
	expected := &mongo.UpdateResult{
		MatchedCount:  0,
		ModifiedCount: 0,
		UpsertedCount: 1,
	}
	require.Equal(t, expected, res)

	// check inserted document
	var doc bson.D
	err = collection.FindOne(ctx, bson.D{{"_id", id}}).Decode(&doc)
	require.NoError(t, err)
	if !AssertEqualDocuments(t, bson.D{{"_id", id}, {"foo", "baz"}}, doc) {
		t.FailNow()
	}

	// this upsert updates document
	filter = bson.D{{"foo", "baz"}}
	update = bson.D{{"$set", bson.D{{"foo", "qux"}}}}
	res, err = collection.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	require.NoError(t, err)

	expected = &mongo.UpdateResult{
		MatchedCount:  1,
		ModifiedCount: 1,
		UpsertedCount: 0,
	}
	require.Equal(t, expected, res)

	// check updated document
	err = collection.FindOne(ctx, bson.D{{"_id", id}}).Decode(&doc)
	require.NoError(t, err)
	AssertEqualDocuments(t, bson.D{{"_id", id}, {"foo", "qux"}}, doc)
}

func TestMultiFlag(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		filter bson.D
		update bson.D
		multi  bool
		stat   bson.D
	}{
		"MultiFalse": {
			filter: bson.D{{"v", int32(42)}},
			update: bson.D{{"$set", bson.D{{"v", int32(43)}}}},
			multi:  false,
			stat:   bson.D{{"n", int32(1)}, {"nModified", int32(1)}, {"ok", float64(1)}},
		},
		"MultiTrue": {
			filter: bson.D{{"v", int32(42)}},
			update: bson.D{{"$set", bson.D{{"v", int32(43)}}}},
			multi:  true,
			stat:   bson.D{{"n", int32(6)}, {"nModified", int32(6)}, {"ok", float64(1)}},
		},
	} {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctx, collection := setup.Setup(t, shareddata.Scalars, shareddata.Composites)

			command := bson.D{
				{"update", collection.Name()},
				{"updates", bson.A{
					bson.D{{"q", tc.filter}, {"u", tc.update}, {"multi", tc.multi}},
				}},
			}

			var result bson.D
			err := collection.Database().RunCommand(ctx, command).Decode(&result)
			require.NoError(t, err)

			AssertEqualDocuments(t, tc.stat, result)
		})
	}
}

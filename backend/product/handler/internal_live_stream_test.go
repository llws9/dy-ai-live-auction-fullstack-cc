package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"product-service/model"
)

type fakeLiveStreamProvider struct {
	items map[int64]*model.LiveStream
	err   error
}

func (f *fakeLiveStreamProvider) GetByIDs(_ context.Context, _ []int64) (map[int64]*model.LiveStream, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.items, nil
}

type fakeUserAvatarProvider struct {
	avatars map[int64]string
	err     error
}

func (f *fakeUserAvatarProvider) GetAvatarsByIDs(_ context.Context, _ []int64) (map[int64]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.avatars, nil
}

type fakeAuctionCountProvider struct {
	counts map[int64]int
	err    error
}

func (f *fakeAuctionCountProvider) CountActiveByLiveStreamIDs(_ context.Context, _ []int64) (map[int64]int, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.counts, nil
}

func newInternalHandlerWithProviders(ls liveStreamBatchProvider, ua userAvatarProvider, ac auctionCountProvider) *InternalHandler {
	return NewInternalHandler(nil, ls, ua, ac)
}

func newInternalHandlerWithLiveStreams(p liveStreamBatchProvider) *InternalHandler {
	return NewInternalHandler(nil, p, nil, nil)
}

func TestInternalHandler_BatchLiveStreams_OK(t *testing.T) {
	h := newInternalHandlerWithProviders(
		&fakeLiveStreamProvider{
			items: map[int64]*model.LiveStream{
				10: {ID: 10, Name: "alice 直播间", CoverImage: "a.jpg", Status: model.LiveStreamStatusActive, CreatorID: 100},
				20: {ID: 20, Name: "bob 直播间", CoverImage: "", Status: model.LiveStreamStatusDisabled, CreatorID: 200},
			},
		},
		&fakeUserAvatarProvider{avatars: map[int64]string{100: "avatar100.png", 200: ""}},
		&fakeAuctionCountProvider{counts: map[int64]int{10: 3}},
	)

	raw, _ := json.Marshal(map[string]interface{}{"ids": []int64{10, 99, 20, 10}})
	c := app.NewContext(0)
	c.Request.SetMethod("POST")
	c.Request.SetRequestURI("/internal/live-streams/batch")
	c.Request.Header.SetContentTypeBytes([]byte("application/json"))
	c.Request.SetBody(raw)

	h.BatchLiveStreams(context.Background(), c)

	assert.Equal(t, 200, c.Response.StatusCode())
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(c.Response.Body(), &resp))
	data := resp["data"].(map[string]interface{})
	items := data["items"].([]interface{})
	require.Len(t, items, 2)

	first := items[0].(map[string]interface{})
	assert.EqualValues(t, 10, first["id"])
	assert.Equal(t, "alice 直播间", first["name"])
	assert.Equal(t, "a.jpg", first["cover_image"])
	assert.EqualValues(t, 1, first["status"])
	assert.EqualValues(t, 100, first["creator_id"])
	assert.Equal(t, "avatar100.png", first["host_avatar"])
	assert.EqualValues(t, 3, first["auction_count"])

	second := items[1].(map[string]interface{})
	assert.EqualValues(t, 20, second["id"])
	assert.EqualValues(t, 0, second["status"])
	assert.Nil(t, second["host_avatar"])
	assert.Nil(t, second["auction_count"])
}

func TestInternalHandler_BatchLiveStreams_HostAvatarAndAuctionCountNullWhenNoProviders(t *testing.T) {
	h := newInternalHandlerWithLiveStreams(&fakeLiveStreamProvider{
		items: map[int64]*model.LiveStream{
			10: {ID: 10, Name: "room", CoverImage: "c.jpg", Status: model.LiveStreamStatusActive, CreatorID: 100},
		},
	})

	raw, _ := json.Marshal(map[string]interface{}{"ids": []int64{10}})
	c := app.NewContext(0)
	c.Request.SetMethod("POST")
	c.Request.SetRequestURI("/internal/live-streams/batch")
	c.Request.Header.SetContentTypeBytes([]byte("application/json"))
	c.Request.SetBody(raw)

	h.BatchLiveStreams(context.Background(), c)

	assert.Equal(t, 200, c.Response.StatusCode())
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(c.Response.Body(), &resp))
	data := resp["data"].(map[string]interface{})
	items := data["items"].([]interface{})
	require.Len(t, items, 1)
	item := items[0].(map[string]interface{})
	assert.Nil(t, item["host_avatar"])
	assert.Nil(t, item["auction_count"])
}

func TestInternalHandler_BatchLiveStreams_AvatarProviderErrorDegradation(t *testing.T) {
	h := newInternalHandlerWithProviders(
		&fakeLiveStreamProvider{
			items: map[int64]*model.LiveStream{
				10: {ID: 10, Name: "room", CoverImage: "c.jpg", Status: model.LiveStreamStatusActive, CreatorID: 100},
			},
		},
		&fakeUserAvatarProvider{err: errors.New("db down")},
		&fakeAuctionCountProvider{counts: map[int64]int{10: 2}},
	)

	raw, _ := json.Marshal(map[string]interface{}{"ids": []int64{10}})
	c := app.NewContext(0)
	c.Request.SetMethod("POST")
	c.Request.SetRequestURI("/internal/live-streams/batch")
	c.Request.Header.SetContentTypeBytes([]byte("application/json"))
	c.Request.SetBody(raw)

	h.BatchLiveStreams(context.Background(), c)

	assert.Equal(t, 200, c.Response.StatusCode())
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(c.Response.Body(), &resp))
	data := resp["data"].(map[string]interface{})
	items := data["items"].([]interface{})
	require.Len(t, items, 1)
	item := items[0].(map[string]interface{})
	assert.Nil(t, item["host_avatar"])
	assert.EqualValues(t, 2, item["auction_count"])
}

func TestInternalHandler_BatchLiveStreams_AuctionCountProviderErrorDegradation(t *testing.T) {
	h := newInternalHandlerWithProviders(
		&fakeLiveStreamProvider{
			items: map[int64]*model.LiveStream{
				10: {ID: 10, Name: "room", CoverImage: "c.jpg", Status: model.LiveStreamStatusActive, CreatorID: 100},
			},
		},
		&fakeUserAvatarProvider{avatars: map[int64]string{100: "a.png"}},
		&fakeAuctionCountProvider{err: errors.New("db down")},
	)

	raw, _ := json.Marshal(map[string]interface{}{"ids": []int64{10}})
	c := app.NewContext(0)
	c.Request.SetMethod("POST")
	c.Request.SetRequestURI("/internal/live-streams/batch")
	c.Request.Header.SetContentTypeBytes([]byte("application/json"))
	c.Request.SetBody(raw)

	h.BatchLiveStreams(context.Background(), c)

	assert.Equal(t, 200, c.Response.StatusCode())
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(c.Response.Body(), &resp))
	data := resp["data"].(map[string]interface{})
	items := data["items"].([]interface{})
	require.Len(t, items, 1)
	item := items[0].(map[string]interface{})
	assert.Equal(t, "a.png", item["host_avatar"])
	assert.Nil(t, item["auction_count"])
}

func TestInternalHandler_BatchLiveStreams_EmptyRejected(t *testing.T) {
	h := newInternalHandlerWithLiveStreams(&fakeLiveStreamProvider{})
	c := app.NewContext(0)
	c.Request.SetMethod("POST")
	c.Request.SetRequestURI("/internal/live-streams/batch")
	c.Request.Header.SetContentTypeBytes([]byte("application/json"))
	c.Request.SetBody([]byte(`{"ids":[]}`))

	h.BatchLiveStreams(context.Background(), c)

	assert.Equal(t, 400, c.Response.StatusCode())
}

func TestInternalHandler_BatchLiveStreams_OversizedRejected(t *testing.T) {
	h := newInternalHandlerWithLiveStreams(&fakeLiveStreamProvider{})
	ids := make([]int64, internalLiveStreamBatchMaxIDs+1)
	for i := range ids {
		ids[i] = int64(i + 1)
	}
	raw, _ := json.Marshal(map[string]interface{}{"ids": ids})
	c := app.NewContext(0)
	c.Request.SetMethod("POST")
	c.Request.SetRequestURI("/internal/live-streams/batch")
	c.Request.Header.SetContentTypeBytes([]byte("application/json"))
	c.Request.SetBody(raw)

	h.BatchLiveStreams(context.Background(), c)

	assert.Equal(t, 400, c.Response.StatusCode())
}

func TestInternalHandler_BatchLiveStreams_DAOErrorReturns500(t *testing.T) {
	h := newInternalHandlerWithLiveStreams(&fakeLiveStreamProvider{err: errors.New("db down")})
	raw, _ := json.Marshal(map[string]interface{}{"ids": []int64{1}})
	c := app.NewContext(0)
	c.Request.SetMethod("POST")
	c.Request.SetRequestURI("/internal/live-streams/batch")
	c.Request.Header.SetContentTypeBytes([]byte("application/json"))
	c.Request.SetBody(raw)

	h.BatchLiveStreams(context.Background(), c)

	assert.Equal(t, 500, c.Response.StatusCode())
}

func TestInternalHandler_BatchLiveStreams_InvalidJSON(t *testing.T) {
	h := newInternalHandlerWithLiveStreams(&fakeLiveStreamProvider{})
	c := app.NewContext(0)
	c.Request.SetMethod("POST")
	c.Request.SetRequestURI("/internal/live-streams/batch")
	c.Request.Header.SetContentTypeBytes([]byte("application/json"))
	c.Request.SetBody(bytes.NewBufferString(`not-json`).Bytes())

	h.BatchLiveStreams(context.Background(), c)

	assert.Equal(t, 400, c.Response.StatusCode())
}

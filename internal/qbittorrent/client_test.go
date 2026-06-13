package qbittorrent

import (
	"net/http"
	"testing"
)

func TestParseAddTorrentResponse_LegacyOk(t *testing.T) {
	result, err := parseAddTorrentResponse(http.StatusOK, []byte("Ok."))
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result for legacy Ok., got %+v", result)
	}
}

func TestParseAddTorrentResponse_LegacyFails(t *testing.T) {
	result, err := parseAddTorrentResponse(http.StatusOK, []byte("Fails."))
	if err == nil {
		t.Fatal("expected error for Fails.")
	}
	if result != nil {
		t.Fatalf("expected nil result on error, got %+v", result)
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.Details != "Fails." {
		t.Fatalf("expected details Fails., got %q", apiErr.Details)
	}
}

func TestParseAddTorrentResponse_EmptyBody(t *testing.T) {
	result, err := parseAddTorrentResponse(http.StatusOK, nil)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result for empty body, got %+v", result)
	}
}

func TestParseAddTorrentResponse_QBittorrent5Success(t *testing.T) {
	body := []byte(`{"added_torrent_ids":["6378ab17f2d70a6756458055ae4727337d1e5b24"],"failure_count":0,"pending_count":0,"success_count":1}`)
	result, err := parseAddTorrentResponse(http.StatusOK, body)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if result == nil {
		t.Fatal("expected parsed result")
	}
	if result.SuccessCount != 1 {
		t.Fatalf("expected success_count 1, got %d", result.SuccessCount)
	}
	if len(result.AddedTorrentIDs) != 1 || result.AddedTorrentIDs[0] != "6378ab17f2d70a6756458055ae4727337d1e5b24" {
		t.Fatalf("unexpected added_torrent_ids: %v", result.AddedTorrentIDs)
	}
}

func TestParseAddTorrentResponse_QBittorrent5Pending(t *testing.T) {
	body := []byte(`{"added_torrent_ids":[],"failure_count":0,"pending_count":1,"success_count":0}`)
	result, err := parseAddTorrentResponse(http.StatusAccepted, body)
	if err != nil {
		t.Fatalf("expected success for pending add, got error: %v", err)
	}
	if result == nil || result.PendingCount != 1 {
		t.Fatalf("expected pending_count 1, got %+v", result)
	}
}

func TestParseAddTorrentResponse_QBittorrent5FailureJSON(t *testing.T) {
	body := []byte(`{"added_torrent_ids":[],"failure_count":1,"pending_count":0,"success_count":0}`)
	result, err := parseAddTorrentResponse(http.StatusOK, body)
	if err == nil {
		t.Fatal("expected error when all adds fail")
	}
	if result != nil {
		t.Fatalf("expected nil result on error, got %+v", result)
	}
}

func TestParseAddTorrentResponse_HTTP409(t *testing.T) {
	body := []byte(`{"added_torrent_ids":[],"failure_count":1,"pending_count":0,"success_count":0}`)
	result, err := parseAddTorrentResponse(http.StatusConflict, body)
	if err == nil {
		t.Fatal("expected error for HTTP 409")
	}
	if result != nil {
		t.Fatalf("expected nil result on error, got %+v", result)
	}
	apiErr, ok := err.(*APIError)
	if !ok || apiErr.Code != http.StatusConflict {
		t.Fatalf("expected 409 APIError, got %+v", err)
	}
}

func TestParseAddTorrentResponse_UnrecognizedBody(t *testing.T) {
	result, err := parseAddTorrentResponse(http.StatusOK, []byte("unknown response"))
	if err == nil {
		t.Fatal("expected error for unrecognized body")
	}
	if result != nil {
		t.Fatalf("expected nil result on error, got %+v", result)
	}
}

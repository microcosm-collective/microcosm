package models

import (
	"net/url"
	"testing"
	"time"
)

func TestSearchQueryParsing(t *testing.T) {
	// Simple query
	u, _ := url.Parse("https://test.microcosm.app/api/v1/search?q=searchTerm&type=event&type=conversation")

	sq := GetSearchQueryFromURL(1, *u, 1)

	if len(sq.ItemTypesQuery) != 2 {
		t.Errorf("Expected 2 itemTypes found %d", len(sq.ItemTypesQuery))
	}

	var foundEvent bool
	var foundConversation bool
	for _, it := range sq.ItemTypesQuery {
		switch it {
		case "conversation":
			foundConversation = true
		case "event":
			foundEvent = true
		default:
			t.Errorf("Found an unknown type: %s", it)
		}
	}

	if !foundConversation {
		t.Error("Did not find the itemType conversation")
	}
	if !foundEvent {
		t.Error("Did not find the itemType event")
	}

	if sq.Query != "searchTerm" {
		t.Errorf("Query does not match: %s", sq.Query)
	}

	// Parse Q
	u, _ = url.Parse("https://test.microcosm.app/api/v1/search?q=searchTerm+type:event+type:conversation")

	sq = GetSearchQueryFromURL(1, *u, 1)

	if len(sq.ItemTypesQuery) != 2 {
		t.Errorf("Expected 2 itemTypes found %d", len(sq.ItemTypesQuery))
	}

	foundEvent = false
	foundConversation = false
	for _, it := range sq.ItemTypesQuery {
		switch it {
		case "conversation":
			foundConversation = true
		case "event":
			foundEvent = true
		default:
			t.Errorf("Found an unknown type: %s", it)
		}
	}

	if !foundConversation {
		t.Error("Did not find the itemType conversation")
	}
	if !foundEvent {
		t.Error("Did not find the itemType event")
	}

	if sq.Query != "searchTerm" {
		t.Errorf("Query does not match: %s", sq.Query)
	}

	// Mix of Q and Query
	u, _ = url.Parse("https://test.microcosm.app/api/v1/search?q=searchTerm+type:event+type:conversation&type=poll")

	sq = GetSearchQueryFromURL(1, *u, 1)

	if len(sq.ItemTypesQuery) != 3 {
		t.Errorf("Expected 3 itemTypes found %d", len(sq.ItemTypesQuery))
	}

	foundEvent = false
	foundConversation = false
	foundPoll := false
	for _, it := range sq.ItemTypesQuery {
		switch it {
		case "conversation":
			foundConversation = true
		case "event":
			foundEvent = true
		case "poll":
			foundPoll = true
		default:
			t.Errorf("Found an unknown type: %s", it)
		}
	}

	if !foundConversation {
		t.Error("Did not find the itemType conversation")
	}
	if !foundEvent {
		t.Error("Did not find the itemType event")
	}
	if !foundPoll {
		t.Error("Did not find the itemType event")
	}

	if sq.Query != "searchTerm" {
		t.Errorf("Query does not match: %s", sq.Query)
	}

	// Mix of Q and Query
	u, _ = url.Parse("https://test.microcosm.app/api/v1/search?q=searchTerm&eventAfter=2012-06-07&type=event")

	sq = GetSearchQueryFromURL(1, *u, 1)

	control, _ := time.Parse("2006-01-02", "2012-06-07")
	if sq.EventAfterTime.Unix() != control.Unix() {
		t.Errorf("Expected %d for 'eventAfter' but was given %d", control.Unix(), sq.EventAfterTime.Unix())
	}

	if sq.Query != "searchTerm" {
		t.Errorf("Query does not match: %s", sq.Query)
	}
}

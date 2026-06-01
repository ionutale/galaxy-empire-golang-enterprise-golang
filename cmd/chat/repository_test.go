package main

import (
	"context"
	"testing"
)

func TestMockRepo_CreateMessage(t *testing.T) {
	repo := newMockRepo()

	msg, err := repo.CreateMessage(context.Background(), "global", 0, 1, "Player 1", "hello")
	if err != nil {
		t.Fatal(err)
	}
	if msg.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if msg.Channel != "global" {
		t.Errorf("expected channel global, got %s", msg.Channel)
	}
	if msg.Content != "hello" {
		t.Errorf("expected content 'hello', got '%s'", msg.Content)
	}
	if msg.SenderID != 1 {
		t.Errorf("expected sender 1, got %d", msg.SenderID)
	}
}

func TestMockRepo_CreateMessage_AutoIncrement(t *testing.T) {
	repo := newMockRepo()

	msg1, _ := repo.CreateMessage(context.Background(), "global", 0, 1, "P1", "first")
	msg2, _ := repo.CreateMessage(context.Background(), "global", 0, 2, "P2", "second")

	if msg2.ID <= msg1.ID {
		t.Errorf("expected msg2.ID > msg1.ID, got %d <= %d", msg2.ID, msg1.ID)
	}
}

func TestMockRepo_GetMessages_Empty(t *testing.T) {
	repo := newMockRepo()

	messages, hasMore, err := repo.GetMessages(context.Background(), "global", 0, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 0 {
		t.Errorf("expected 0 messages, got %d", len(messages))
	}
	if hasMore {
		t.Error("expected hasMore false")
	}
}

func TestMockRepo_GetMessages_ChannelFilter(t *testing.T) {
	repo := newMockRepo()

	repo.CreateMessage(context.Background(), "global", 0, 1, "P1", "global msg")
	repo.CreateMessage(context.Background(), "alliance", 42, 2, "P2", "alliance msg")
	repo.CreateMessage(context.Background(), "global", 0, 3, "P3", "another global")

	globalMsgs, _, _ := repo.GetMessages(context.Background(), "global", 0, 10, 0)
	if len(globalMsgs) != 2 {
		t.Errorf("expected 2 global messages, got %d", len(globalMsgs))
	}

	allianceMsgs, _, _ := repo.GetMessages(context.Background(), "alliance", 42, 10, 0)
	if len(allianceMsgs) != 1 {
		t.Errorf("expected 1 alliance message, got %d", len(allianceMsgs))
	}
}

func TestMockRepo_GetMessages_ChannelIDFilter(t *testing.T) {
	repo := newMockRepo()

	repo.CreateMessage(context.Background(), "alliance", 42, 1, "P1", "alliance 42")
	repo.CreateMessage(context.Background(), "alliance", 99, 2, "P2", "alliance 99")

	msgs1, _, _ := repo.GetMessages(context.Background(), "alliance", 42, 10, 0)
	if len(msgs1) != 1 || msgs1[0].Content != "alliance 42" {
		t.Errorf("expected 1 message for alliance 42, got %d", len(msgs1))
	}
}

func TestMockRepo_GetMessages_Limit(t *testing.T) {
	repo := newMockRepo()

	for i := 0; i < 5; i++ {
		repo.CreateMessage(context.Background(), "global", 0, 1, "P1", "msg")
	}

	messages, hasMore, err := repo.GetMessages(context.Background(), "global", 0, 3, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 3 {
		t.Errorf("expected 3 messages, got %d", len(messages))
	}
	if !hasMore {
		t.Error("expected hasMore true")
	}
}

func TestMockRepo_GetMessages_BeforeID(t *testing.T) {
	repo := newMockRepo()

	var ids []int
	for i := 0; i < 5; i++ {
		msg, _ := repo.CreateMessage(context.Background(), "global", 0, 1, "P1", "msg")
		ids = append(ids, msg.ID)
	}

	messages, hasMore, err := repo.GetMessages(context.Background(), "global", 0, 10, ids[3])
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 3 {
		t.Errorf("expected 3 messages before id %d, got %d", ids[3], len(messages))
	}
	if hasMore {
		t.Error("expected hasMore false when before_id")
	}
}

func TestMockRepo_GetMessages_ChronologicalOrder(t *testing.T) {
	repo := newMockRepo()

	repo.CreateMessage(context.Background(), "global", 0, 1, "P1", "first")
	repo.CreateMessage(context.Background(), "global", 0, 2, "P2", "second")
	repo.CreateMessage(context.Background(), "global", 0, 3, "P3", "third")

	messages, _, _ := repo.GetMessages(context.Background(), "global", 0, 10, 0)

	if len(messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(messages))
	}
	if messages[0].Content != "first" {
		t.Errorf("expected first message 'first', got '%s'", messages[0].Content)
	}
	if messages[2].Content != "third" {
		t.Errorf("expected last message 'third', got '%s'", messages[2].Content)
	}
}

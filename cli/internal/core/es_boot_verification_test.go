package core

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"google.golang.org/protobuf/proto"
	corev1 "hmans.de/chatto/internal/pb/chatto/core/v1"
	"hmans.de/chatto/internal/testutil"
)

func TestEvaluateESBootVerificationReportDetectsCountRegressions(t *testing.T) {
	report := &esBootVerificationReport{
		legacy: esLegacyCounts{
			rooms:               3,
			memberships:         7,
			roomGroups:          2,
			roomLayoutPresent:   true,
			serverConfigPresent: true,
			messages:            9,
			threadReplies:       2,
			reactions:           4,
		},
		projected: esProjectedCounts{
			rooms:                 3,
			memberships:           6,
			roomGroups:            2,
			roomLayoutGroups:      0,
			serverConfigProjected: false,
			messagePosts:          8,
			threadReplies:         0,
			activeReactions:       4,
		},
		decodeErrors: 1,
	}

	var core ChattoCore
	core.evaluateESBootVerificationReport(report)

	assertProblemContains(t, report.problems, "memberships")
	assertProblemContains(t, report.problems, "messages")
	assertProblemContains(t, report.problems, "thread replies")
	assertProblemContains(t, report.problems, "server config")
	assertProblemContains(t, report.problems, "room layout")
	assertProblemContains(t, report.problems, "decode errors")
}

func TestEvaluateESBootVerificationReportAllowsProjectedSuperset(t *testing.T) {
	report := &esBootVerificationReport{
		legacy: esLegacyCounts{
			rooms:         2,
			memberships:   3,
			roomGroups:    1,
			messages:      5,
			threadReplies: 2,
			reactions:     1,
		},
		projected: esProjectedCounts{
			rooms:           3,
			memberships:     4,
			roomGroups:      2,
			messagePosts:    6,
			threadReplies:   2,
			activeReactions: 1,
		},
	}

	var core ChattoCore
	core.evaluateESBootVerificationReport(report)

	if len(report.problems) != 0 {
		t.Fatalf("problems = %v, want none", report.problems)
	}
}

func TestCountLegacyRoomMembershipPairsDedupesKeyShapes(t *testing.T) {
	_, nc := testutil.StartNATS(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatalf("jetstream: %v", err)
	}
	kv, err := js.CreateOrUpdateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket:  "SERVER_CONFIG_TEST",
		Storage: jetstream.MemoryStorage,
	})
	if err != nil {
		t.Fatalf("create KV: %v", err)
	}

	data, err := proto.Marshal(&corev1.RoomMembership{})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, key := range []string{
		"room_membership.channel.R1.U1",
		"room_membership.U1.R1",
		"room_membership.channel.R1.U2",
	} {
		if _, err := kv.Put(ctx, key, data); err != nil {
			t.Fatalf("put %s: %v", key, err)
		}
	}

	unique, raw, malformed, err := countLegacyRoomMembershipPairs(ctx, kv)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if unique != 2 || raw != 3 || malformed != 0 {
		t.Fatalf("countLegacyRoomMembershipPairs = unique %d raw %d malformed %d, want 2, 3, 0", unique, raw, malformed)
	}
}

func TestCountLegacyMessagePostedEventsIgnoresNilPayloadRecords(t *testing.T) {
	_, nc := testutil.StartNATS(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatalf("jetstream: %v", err)
	}
	stream, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     "SERVER_EVENTS_TEST",
		Subjects: []string{"server.>"},
		Storage:  jetstream.MemoryStorage,
	})
	if err != nil {
		t.Fatalf("create stream: %v", err)
	}

	publish := func(subject string, event *corev1.Event) {
		t.Helper()
		data, err := proto.Marshal(event)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if _, err := js.Publish(ctx, subject, data); err != nil {
			t.Fatalf("publish: %v", err)
		}
	}
	publish("server.room.channel.R1.msg.M1", &corev1.Event{
		Id: "M1",
		Event: &corev1.Event_MessagePosted{
			MessagePosted: &corev1.MessagePostedEvent{RoomId: "R1"},
		},
	})
	publish("server.room.channel.R1.msg.EMPTY", &corev1.Event{Id: "EMPTY"})

	posts, nonPosts, decodeErrors, err := countLegacyMessagePostedEvents(ctx, stream, []string{"server.room.*.*.msg.>"})
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if posts != 1 || nonPosts != 1 || decodeErrors != 0 {
		t.Fatalf("countLegacyMessagePostedEvents = posts %d nonPosts %d decodeErrors %d, want 1, 1, 0", posts, nonPosts, decodeErrors)
	}
}

func assertProblemContains(t *testing.T, problems []string, needle string) {
	t.Helper()
	for _, problem := range problems {
		if strings.Contains(problem, needle) {
			return
		}
	}
	t.Fatalf("problems %v did not contain %q", problems, needle)
}

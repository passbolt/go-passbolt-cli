package group

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/passbolt/go-passbolt/helper"
)

func TestGroupDetailExposesMembershipAndUserIDs(t *testing.T) {
	out := captureOutput(t, func() {
		if err := printGroupDetail("group-id", "team", []helper.GroupMembership{{MembershipID: "membership-id", UserID: "user-id", Username: "ada@example.com", IsGroupManager: false}}, nil, true); err != nil {
			t.Fatal(err)
		}
	})
	var got struct {
		ID        string           `json:"id"`
		UserCount int              `json:"user_count"`
		Users     []map[string]any `json:"users"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != "group-id" || got.UserCount != 1 || len(got.Users) != 1 {
		t.Fatalf("output=%s", out)
	}
	user := got.Users[0]
	if user["id"] != "user-id" || user["membership_id"] != "membership-id" || user["is_group_manager"] != false {
		t.Errorf("user=%v", user)
	}
}

func TestGroupDetailMembershipIDColumn(t *testing.T) {
	out := captureOutput(t, func() {
		if err := printGroupDetail("group-id", "team", []helper.GroupMembership{{MembershipID: "membership-id", UserID: "user-id"}}, []string{"MembershipID", "UserID"}, false); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out, "membership-id") || !strings.Contains(out, "user-id") {
		t.Errorf("output=%s", out)
	}
}

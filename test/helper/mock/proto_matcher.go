package mock

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
)

type ProtoMessageMatcher struct {
	msg proto.Message
}

func (m *ProtoMessageMatcher) Matches(x interface{}) bool {
	msg, ok := x.(proto.Message)
	if !ok {
		return false
	}
	return proto.Equal(msg, m.msg)
}

func (m *ProtoMessageMatcher) Got(got interface{}) string {
	return fmt.Sprintf("%v (%T)\nDiff (-want +got):\n%s", got, got, strings.TrimSpace(cmp.Diff(m.msg, got, protocmp.Transform())))
}

func (m *ProtoMessageMatcher) String() string {
	raw, _ := json.Marshal(m.msg)
	return fmt.Sprintf("%s (%T)\n", string(raw), m.msg)
}

type ProtoSliceMatcher struct {
	msgs interface{}
}

func (m *ProtoSliceMatcher) Matches(xs interface{}) bool {
	s := reflect.ValueOf(xs)
	if s.Kind() != reflect.Slice {
		return false
	}

	msgs := reflect.ValueOf(m.msgs)
	if msgs.Kind() != reflect.Slice {
		return false
	}

	if msgs.Len() != s.Len() {
		return false
	}

	visited := make([]bool, s.Len())
	for i := 0; i < s.Len(); i++ {
		found := false

		for j := 0; j < msgs.Len(); j++ {
			if visited[j] {
				continue
			}
			msg, ok := s.Index(i).Interface().(proto.Message)
			if !ok {
				return false
			}
			src, ok := msgs.Index(j).Interface().(proto.Message)
			if !ok {
				return false
			}
			if !proto.Equal(msg, src) {
				continue
			}
			found = true
			visited[j] = true
			break
		}
		if !found {
			return false
		}
	}

	return true
}

func (m *ProtoSliceMatcher) Got(got interface{}) string {
	return fmt.Sprintf("%v (%T)\nDiff (-want +got):\n%s", got, got, strings.TrimSpace(cmp.Diff(m.msgs, got, protocmp.Transform())))
}

func (m *ProtoSliceMatcher) String() string {
	raw, _ := json.Marshal(m.msgs)
	return fmt.Sprintf("%s (%T)\n", string(raw), m.msgs)
}

func ProtoEq(msg proto.Message) *ProtoMessageMatcher {
	return &ProtoMessageMatcher{msg: msg}
}

func ProtoSliceEq(msgs interface{}) *ProtoSliceMatcher {
	return &ProtoSliceMatcher{msgs: msgs}
}

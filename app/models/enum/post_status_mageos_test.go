package enum_test

import (
	"testing"

	. "github.com/getfider/fider/app/pkg/assert"

	"github.com/getfider/fider/app/models/enum"
)

func TestPostUnderReview_RoundTrip(t *testing.T) {
	RegisterT(t)

	text, err := enum.PostUnderReview.MarshalText()
	Expect(err).IsNil()
	Expect(string(text)).Equals("underreview")

	var status enum.PostStatus
	Expect(status.UnmarshalText([]byte("underreview"))).IsNil()
	Expect(status).Equals(enum.PostUnderReview)
}

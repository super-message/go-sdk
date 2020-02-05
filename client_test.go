package go_sdk

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestMemoryCache(t *testing.T) {
	cache := NewMemoryCache()

	Convey("Add a value to cache ", t, func() {
		m := Member{
			OpenID:         "1",
			ChannelCreator: true,
			ExpiredAt:      time.Now().Unix() + 1,
		}
		_ = cache.Set("a", m)

		Convey("Getting value from cache", func() {
			m2, ok := cache.Get("a")
			So(ok, ShouldBeTrue)
			So(m2.OpenID, ShouldEqual, m.OpenID)
			So(m2.ChannelCreator, ShouldEqual, m.ChannelCreator)
		})

		time.Sleep(time.Second)
		Convey("Getting value from cache when value expired", func() {
			_, ok := cache.Get("a")
			So(ok, ShouldBeFalse)
		})
	})
}

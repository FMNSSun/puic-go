package quic

import (
	"github.com/lucas-clemente/quic-go/frames"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Stream", func() {
	It("reads a single StreamFrame", func() {
		frame := frames.StreamFrame{
			Offset: 0,
			Data:   []byte{0xDE, 0xAD, 0xBE, 0xEF},
		}
		stream := NewStream()
		stream.AddStreamFrame(&frame)
		b := make([]byte, 4)
		n, err := stream.Read(b)
		Expect(err).ToNot(HaveOccurred())
		Expect(n).To(Equal(4))
		Expect(stream.DataLen).To(Equal(uint64(4)))
		Expect(b).To(Equal([]byte{0xDE, 0xAD, 0xBE, 0xEF}))
	})

	It("assembles multiple StreamFrames", func() {
		frame1 := frames.StreamFrame{
			Offset: 0,
			Data:   []byte{0xDE, 0xAD},
		}
		frame2 := frames.StreamFrame{
			Offset: 2,
			Data:   []byte{0xBE, 0xEF},
		}
		stream := NewStream()
		stream.AddStreamFrame(&frame1)
		stream.AddStreamFrame(&frame2)
		b := make([]byte, 4)
		n, err := stream.Read(b)
		Expect(err).ToNot(HaveOccurred())
		Expect(n).To(Equal(4))
		Expect(stream.DataLen).To(Equal(uint64(4)))
		Expect(b).To(Equal([]byte{0xDE, 0xAD, 0xBE, 0xEF}))
	})

	It("rejects StreamFrames with wrong Offsets", func() {
		frame1 := frames.StreamFrame{
			Offset: 0,
			Data:   []byte{0xDE, 0xAD},
		}
		frame2 := frames.StreamFrame{
			Offset: 1,
			Data:   []byte{0xBE, 0xEF},
		}
		stream := NewStream()
		stream.AddStreamFrame(&frame1)
		err := stream.AddStreamFrame(&frame2)
		Expect(err).To(HaveOccurred())
	})
})
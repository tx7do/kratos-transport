package webtransport

import (
	"io"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/tx7do/kratos-transport/transport/webtransport/mock"
)

//go:generate mockgen -package mock -destination mock/stream_creator_mock.go github.com/quic-go/quic-go/http3 StreamCreator
////go:generate sh -c "mockgen -package webtransport -destination mock_stream_test.go github.com/quic-go/quic-go Stream && cat mock_stream_test.go | sed s@protocol\\.StreamID@quic.StreamID@g | sed s@qerr\\.StreamErrorCode@quic.StreamErrorCode@g > tmp.go && mv tmp.go mock_stream_test.go && goimports -w mock_stream_test.go"

type mockRequestStream struct {
	*mock.MockStream
	c chan struct{}
}

func newMockRequestStream(ctrl *gomock.Controller) quic.Stream {
	str := mock.NewMockStream(ctrl)
	str.EXPECT().Close()
	str.EXPECT().CancelRead(gomock.Any())
	return &mockRequestStream{MockStream: str, c: make(chan struct{})}
}

var _ io.ReadWriteCloser = &mockRequestStream{}

func (s *mockRequestStream) Close() error {
	s.MockStream.Close()
	close(s.c)
	return nil
}

func (s *mockRequestStream) Read(b []byte) (int, error)  { <-s.c; return 0, io.EOF }
func (s *mockRequestStream) Write(b []byte) (int, error) { return len(b), nil }

func TestCloseStreamsOnClose(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSess := mock.NewMockStreamCreator(ctrl)
	sess := newSession(42, mockSess, newMockRequestStream(ctrl))

	str := mock.NewMockStream(ctrl)
	str.EXPECT().StreamID().Return(quic.StreamID(4)).AnyTimes()
	mockSess.EXPECT().OpenStream().Return(str, nil)
	_, err := sess.OpenStream()
	require.NoError(t, err)
	ustr := mock.NewMockStream(ctrl)
	ustr.EXPECT().StreamID().Return(quic.StreamID(5)).AnyTimes()
	mockSess.EXPECT().OpenUniStream().Return(ustr, nil)
	_, err = sess.OpenUniStream()
	require.NoError(t, err)

	str.EXPECT().CancelRead(sessionCloseErrorCode)
	str.EXPECT().CancelWrite(sessionCloseErrorCode)
	ustr.EXPECT().CancelWrite(sessionCloseErrorCode)
	require.NoError(t, sess.Close())
}

func TestAddStreamAfterSessionClose(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sess := newSession(42, mock.NewMockStreamCreator(ctrl), newMockRequestStream(ctrl))
	require.NoError(t, sess.Close())

	str := mock.NewMockStream(ctrl)
	str.EXPECT().CancelRead(sessionCloseErrorCode)
	str.EXPECT().CancelWrite(sessionCloseErrorCode)
	sess.addIncomingStream(str)

	ustr := mock.NewMockStream(ctrl)
	ustr.EXPECT().CancelRead(sessionCloseErrorCode)
	sess.addIncomingUniStream(ustr)
}

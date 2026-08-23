package kafka // nolint: testpackage

import (
	"errors"
	"testing"

	smocks "github.com/IBM/sarama/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testTopic = "test-avatar-topic"

// setupProducerTest is a helper to initialize the Producer with a Sarama mock.
func setupProducerTest(t *testing.T) (*Producer, *smocks.SyncProducer) {
	t.Helper()
	mockProducer := smocks.NewSyncProducer(t, nil)

	p := &Producer{
		producer: mockProducer,
		topic:    testTopic,
	}

	return p, mockProducer
}

func TestProducer_PublishAvatarCreatedEvent_Success(t *testing.T) {
	producer, mockProducer := setupProducerTest(t)

	// Tell the mock to expect a message and succeed
	mockProducer.ExpectSendMessageAndSucceed()

	err := producer.PublishAvatarCreatedEvent(t.Context(), "avatar-123", "user-456", "avatars/user-456/avatar-123")

	require.NoError(t, err)

	// Close validates that all expectations were met
	assert.NoError(t, mockProducer.Close())
}

func TestProducer_PublishAvatarCreatedEvent_KafkaError(t *testing.T) {
	producer, mockProducer := setupProducerTest(t)

	simulatedError := errors.New("kafka broker is unavailable")
	mockProducer.ExpectSendMessageAndFail(simulatedError)

	err := producer.PublishAvatarCreatedEvent(t.Context(), "avatar-123", "user-456", "avatars/user-456/avatar-123")

	require.Error(t, err)
	assert.ErrorIs(t, err, simulatedError)
	assert.Contains(t, err.Error(), "sending message to kafka")

	assert.NoError(t, mockProducer.Close())
}

func TestProducer_Close_Success(t *testing.T) {
	producer, _ := setupProducerTest(t)

	err := producer.Close()

	require.NoError(t, err)
	// Mock is already closed by producer.Close(), no need to call mockProducer.Close() again
}

package direct

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockDriver struct {
	id      string
	support []DriverMessage
}

func (m mockDriver) ID() string {
	return m.id
}

func (m mockDriver) Description() string {
	return ""
}

func (m mockDriver) FID() string {
	return ""
}

func (m mockDriver) Org() string {
	return ""
}

func (m mockDriver) URL() string {
	return ""
}

func (m mockDriver) MessageSupport() []DriverMessage {
	return m.support
}

func driverCleanup() {
	directConnectInstitutions = make(map[string]Driver)
}

func TestRegister(t *testing.T) {
	t.Run("supported driver", func(t *testing.T) {
		defer driverCleanup()
		assert.Len(t, directConnectInstitutions, 0)
		driver := mockDriver{id: "mock ID", support: []DriverMessage{MessageBank}}
		Register(driver)
		assert.Len(t, directConnectInstitutions, 1)
		assert.Equal(t, driver, directConnectInstitutions["mock ID"])
	})

	t.Run("unsupported driver", func(t *testing.T) {
		defer driverCleanup()
		assert.Len(t, directConnectInstitutions, 0)
		driver := mockDriver{id: "mock ID"}
		Register(driver)
		assert.Len(t, directConnectInstitutions, 0)
	})
}

func TestSearch(t *testing.T) {
	defer driverCleanup()
	driver := mockDriver{id: "mock ID", support: []DriverMessage{MessageBank}}
	directConnectInstitutions["mock ID"] = driver
	assert.Equal(t, []Driver{driver}, Search(""))
	assert.Equal(t, []Driver{}, Search("foo"))
}

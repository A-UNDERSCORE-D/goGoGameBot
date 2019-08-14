package format

import "sync"

// Storage holds data for cross-execution storage
type Storage struct {
	sync.RWMutex // Lets not have lists explode
	data         map[string]interface{}
}

func (s *Storage) checkMap() {
	s.Lock()
	if s.data == nil {
		s.data = make(map[string]interface{})
	}
	s.Unlock()
}

func (s *Storage) set(name string, data interface{}) {
	s.checkMap()
	s.Lock()
	s.data[name] = data
	s.Unlock()
}

func (s *Storage) get(name string) (interface{}, bool) {
	s.checkMap()
	s.RLock()
	defer s.RUnlock()
	data, ok := s.data[name]
	return data, ok // Cant just return data[testName] apparently
}

// SetInt sets the int at `testName` to `data`
func (s *Storage) SetInt(name string, i int) int {
	s.set(name, i)
	return i
}

// GetInt returns either the int stored at `testName` or a default
func (s *Storage) GetInt(name string, def int) int {
	if data, ok := s.get(name); ok {
		return data.(int)
	}
	return def
}

// SetBool sets the bool at `testName` to `data`
func (s *Storage) SetBool(name string, data bool) bool {
	s.set(name, data)
	return data
}

// GetBool returns either the bool stored at `testName` or a default
func (s *Storage) GetBool(name string, def bool) bool {
	if res, ok := s.get(name); ok {
		return res.(bool)
	}
	return def
}

// SetString sets the string at `testName` to `data`
func (s *Storage) SetString(name, data string) string {
	s.set(name, data)
	return data
}

// GetString returns either the string stored at `testName` or a default
func (s *Storage) GetString(name, def string) string {
	if res, ok := s.get(name); ok {
		return res.(string)
	}
	return def
}

// Delete deletes an entry from the Storage, It will not error if the entry does not exist
func (s *Storage) Delete(name string) string {
	s.Lock()
	delete(s.data, name)
	s.Unlock()
	return name
}

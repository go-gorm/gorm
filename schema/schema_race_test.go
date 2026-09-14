package schema

import (
	"reflect"
	"sync"
	"testing"
	"time"
)

type getOrParseRaceModel struct {
	ID uint
}

type getOrParseEmbeddedRaceModel struct {
	getOrParseRaceModel `gorm:"embedded"`
}

func TestGetOrParse_WaitsForCachedSchemaInitialization(t *testing.T) {
	cacheStore := &sync.Map{}
	modelType := reflect.TypeOf(getOrParseRaceModel{})

	initialized := make(chan struct{})
	cachedSchema := &Schema{
		Name:        modelType.Name(),
		ModelType:   modelType,
		Table:       "get_or_parse_race_models",
		initialized: initialized,
		Relationships: Relationships{
			Relations: map[string]*Relationship{},
		},
	}

	cacheStore.Store(modelType, cachedSchema)

	done := make(chan *Schema, 1)
	go func() {
		s, err := getOrParse(getOrParseRaceModel{}, cacheStore, NamingStrategy{})
		if err != nil {
			t.Errorf("getOrParse returned error: %v", err)
			return
		}
		done <- s
	}()

	select {
	case s := <-done:
		t.Fatalf("getOrParse returned schema %v before initialized was closed", s)
	case <-time.After(50 * time.Millisecond):
	}

	cachedSchema.Relationships.Relations["Ready"] = &Relationship{Name: "Ready"}
	close(initialized)

	select {
	case s := <-done:
		if s != cachedSchema {
			t.Fatalf("getOrParse returned unexpected schema pointer")
		}
		if _, ok := s.Relationships.Relations["Ready"]; !ok {
			t.Fatalf("getOrParse returned before cached schema was fully populated")
		}
	case <-time.After(time.Second):
		t.Fatal("getOrParse did not return after initialized was closed")
	}
}

func TestGetOrParse_AllowsCurrentParseChainToAvoidSelfDeadlock(t *testing.T) {
	cacheStore := &sync.Map{}
	modelType := reflect.TypeOf(getOrParseRaceModel{})

	initialized := make(chan struct{})
	cachedSchema := &Schema{
		Name:        modelType.Name(),
		ModelType:   modelType,
		Table:       "get_or_parse_race_models",
		initialized: initialized,
		Relationships: Relationships{
			Relations: map[string]*Relationship{},
		},
	}

	cacheStore.Store(modelType, cachedSchema)

	done := make(chan *Schema, 1)
	go func() {
		s, err := getOrParseWithCallers(
			getOrParseRaceModel{},
			cacheStore,
			NamingStrategy{},
			map[reflect.Type]struct{}{modelType: {}},
		)
		if err != nil {
			t.Errorf("getOrParseWithCallers returned error: %v", err)
			return
		}
		done <- s
	}()

	select {
	case s := <-done:
		if s != cachedSchema {
			t.Fatalf("getOrParseWithCallers returned unexpected schema pointer")
		}
	case <-time.After(time.Second):
		t.Fatal("getOrParseWithCallers waited for a schema in the current parse chain")
	}

	close(initialized)
}

func TestGetOrParse_EmbeddedSelfDeadlock(t *testing.T) {
	cacheStore := &sync.Map{}
	modelType := reflect.TypeOf(getOrParseRaceModel{})

	initialized := make(chan struct{})
	cachedSchema := &Schema{
		Name:        modelType.Name(),
		ModelType:   modelType,
		Table:       "get_or_parse_race_models",
		initialized: initialized,
		Relationships: Relationships{
			Relations: map[string]*Relationship{},
		},
	}

	cacheStore.Store(modelType, cachedSchema)

	done := make(chan struct{})
	go func() {
		_, err := getOrParse(getOrParseEmbeddedRaceModel{}, cacheStore, NamingStrategy{})
		if err != nil {
			t.Errorf("getOrParse returned error: %v", err)
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("getOrParse deadlocked on embedded model")
	}

	close(initialized)
}

type getOrParsePet struct {
	ID     uint
	UserID uint
	User   *getOrParseUser `gorm:"foreignKey:UserID"`
}

type getOrParseUser struct {
	ID   uint
	Pets []*getOrParsePet `gorm:"foreignKey:UserID"`
}

func TestGetOrParse_CyclicDeadlock(t *testing.T) {
	cacheStore := &sync.Map{}
	namer := NamingStrategy{}

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, err := getOrParse(&getOrParseUser{}, cacheStore, namer)
		if err != nil {
			t.Errorf("getOrParse returned error: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		_, err := getOrParse(&getOrParsePet{}, cacheStore, namer)
		if err != nil {
			t.Errorf("getOrParse returned error: %v", err)
		}
	}()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("getOrParse deadlocked on cyclic models")
	}
}

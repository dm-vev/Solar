package event

// Level event data types and global event instances.

// LevelSaveData fires before the world is saved.
// Cancelable — if cancelled, the save is skipped.
type LevelSaveData struct{}

// LevelLoadData fires before a level is loaded.
// Cancelable — if cancelled, the load is aborted.
type LevelLoadData struct {
	Name string
}

// LevelLoadedData — level has been loaded
type LevelLoadedData struct{ Name string }

// LevelAddedData — level added to loaded list
type LevelAddedData struct{ Name string }

// LevelRemovedData — level removed from loaded list
type LevelRemovedData struct{ Name string }

// LevelUnloadData — level about to unload. Cancelable.
type LevelUnloadData struct{ Name string }

// LevelDeletedData — level deleted
type LevelDeletedData struct{ Name string }

// LevelCopiedData — level copied
type LevelCopiedData struct {
	Source string
	Dest   string
}

// LevelRenamedData — level renamed
type LevelRenamedData struct {
	Source string
	Dest   string
}

// MainLevelChangingData — main level is changing. Modifiable.
type MainLevelChangingData struct{ Map *string }

// PhysicsUpdateData fires when a physics block update occurs.
type PhysicsUpdateData struct {
	X, Y, Z int
	Block   byte
	Level   string
}

// PhysicsStateChangedData fires when physics mode changes.
type PhysicsStateChangedData struct {
	Level string
	Mode  int
}

var (
	OnLevelSave         = NewEvent[LevelSaveData]()
	OnLevelLoad         = NewEvent[LevelLoadData]()
	OnLevelLoaded       = NewEvent[LevelLoadedData]()
	OnLevelAdded        = NewEvent[LevelAddedData]()
	OnLevelRemoved      = NewEvent[LevelRemovedData]()
	OnLevelUnload       = NewEvent[LevelUnloadData]()
	OnLevelDeleted      = NewEvent[LevelDeletedData]()
	OnLevelCopied       = NewEvent[LevelCopiedData]()
	OnLevelRenamed      = NewEvent[LevelRenamedData]()
	OnMainLevelChanging = NewEvent[MainLevelChangingData]()

	OnPhysicsUpdate       = NewEvent[PhysicsUpdateData]()
	OnPhysicsStateChanged = NewEvent[PhysicsStateChangedData]()
)

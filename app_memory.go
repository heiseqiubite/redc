package main

import (
	redc "red-cloud/mod"
)

// GetAgentMemories returns all agent memories for the current project
func (a *App) GetAgentMemories() ([]redc.MemoryItem, error) {
	a.mu.Lock()
	project := a.project
	a.mu.Unlock()
	if project == nil {
		return nil, nil
	}

	if a.memoryStore == nil {
		return nil, nil
	}

	items, err := a.memoryStore.ListMemories(project.ProjectName)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []redc.MemoryItem{}
	}
	return items, nil
}

// DeleteAgentMemory deletes a specific agent memory by ID
func (a *App) DeleteAgentMemory(id int) error {
	if a.memoryStore == nil {
		return nil
	}
	return a.memoryStore.DeleteMemory(id)
}

// ClearAgentMemories clears all agent memories for the current project
func (a *App) ClearAgentMemories() error {
	a.mu.Lock()
	project := a.project
	a.mu.Unlock()
	if project == nil {
		return nil
	}
	if a.memoryStore == nil {
		return nil
	}
	return a.memoryStore.ClearMemories(project.ProjectName)
}

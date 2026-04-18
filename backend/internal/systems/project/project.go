package project

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type Project struct {
	Path         string    `json:"path"`
	Sessions     []string  `json:"sessions"`
	LastModified time.Time `json:"last_modified"`
}

type ProjectManager struct {
	configFile string
	projects   map[string]*Project
	mu         sync.RWMutex
}

func NewProjectManager(projectRoot string) *ProjectManager {
	configFile := filepath.Join(projectRoot, ".projects/", "projects.json")

	pm := &ProjectManager{
		configFile: configFile,
		projects:   make(map[string]*Project),
	}
	pm.load()
	return pm
}

func (pm *ProjectManager) load() {
	data, err := os.ReadFile(pm.configFile)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Failed to read project config: %v", err)
		}
		return
	}

	if len(data) == 0 {
		return
	}

	var container struct {
		Projects map[string]*Project `json:"projects"`
	}
	if err := json.Unmarshal(data, &container); err != nil {
		log.Printf("Warning: Failed to parse projects.json: %v", err)
		return
	}
	if container.Projects != nil {
		pm.projects = container.Projects
		for _, p := range pm.projects {
			if p.LastModified.IsZero() {
				p.LastModified = time.Now()
			}
		}
	}
}

func (pm *ProjectManager) save() {
	pm.mu.RLock()
	container := struct {
		Projects map[string]*Project `json:"projects"`
	}{
		Projects: pm.projects,
	}
	pm.mu.RUnlock()

	data, _ := json.MarshalIndent(container, "", "  ")
	os.MkdirAll(filepath.Dir(pm.configFile), 0755)
	os.WriteFile(pm.configFile, data, 0644)
}

func (pm *ProjectManager) GetOrCreate(path string) *Project {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	key := filepath.Base(path)
	if project, exists := pm.projects[key]; exists {
		return project
	}

	project := &Project{
		Path:         path,
		Sessions:     []string{},
		LastModified: time.Now(),
	}
	pm.projects[key] = project
	pm.save()
	return project
}

func (pm *ProjectManager) OpenProject(path string) *Project {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	key := filepath.Base(path)
	if project, exists := pm.projects[key]; exists {
		project.LastModified = time.Now()
		pm.save()
		return project
	}

	project := &Project{
		Path:         path,
		Sessions:     []string{},
		LastModified: time.Now(),
	}
	pm.projects[key] = project
	pm.save()
	return project
}

func (pm *ProjectManager) AddSession(projectPath, sessionID string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	key := filepath.Base(projectPath)
	project, exists := pm.projects[key]
	if !exists {
		project = &Project{
			Path:         projectPath,
			Sessions:     []string{},
			LastModified: time.Now(),
		}
		pm.projects[key] = project
	}

	for _, id := range project.Sessions {
		if id == sessionID {
			return
		}
	}
	project.Sessions = append(project.Sessions, sessionID)
	project.LastModified = time.Now()
	pm.save()
}

func (pm *ProjectManager) GetProject(path string) *Project {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	key := filepath.Base(path)
	return pm.projects[key]
}

func (pm *ProjectManager) ListProjects() []*Project {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var list []*Project
	for _, p := range pm.projects {
		list = append(list, p)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].LastModified.After(list[j].LastModified)
	})
	return list
}

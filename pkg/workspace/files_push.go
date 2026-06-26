package workspace

// Push copies files from a host source to the workspace.
func (m *Manager) Push(hostSrc, relPath string) error {
	dstAbs, err := m.ValidatePath(relPath)
	if err != nil {
		return err
	}
	return copyRecursive(hostSrc, dstAbs)
}

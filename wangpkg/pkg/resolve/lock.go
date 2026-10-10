package resolve

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/startracex/wangpkg/pkg/lockfile"
	"github.com/startracex/wangpkg/pkg/specifier"
	"github.com/startracex/wangpkg/pkg/workspace"
)

func formatVersion(request, resolved specifier.Specifier) string {
	if request.Type() != resolved.Type() {
		return resolved.String()
	}
	switch resolved := resolved.(type) {
	case *specifier.RegistrySpecifier:
		request := request.(*specifier.RegistrySpecifier)
		newVersion := loosePinning(request.Version, resolved.Version)
		if request.Scheme == "npm" && resolved.Scheme == "npm" && request.Alias() == resolved.Alias() {
			return newVersion
		}
		return resolved.WithVersion(newVersion).String()
	case *specifier.WorkspaceSpecifier:
		if request.Alias() == resolved.Alias() {
			return "workspace:" + resolved.Version
		}
		return resolved.String()
	default:
		return resolved.String()
	}
}

func loosePinning(request, resolvedSemver string) string {
	// constraint ^
	if request == "" || (strings.HasPrefix(request, "^") && isVersion(request[1:])) {
		return "^" + resolvedSemver
	}

	// constraint ~
	if strings.HasPrefix(request, "~") && isVersion(request[1:]) {
		return "~" + resolvedSemver
	}

	// strict
	if isVersion(request) {
		return request
	}

	// constraint
	if isConstraint(resolvedSemver) {
		return request
	}

	// dist tag
	return "^" + resolvedSemver
}

func isVersion(s string) bool {
	_, err := semver.NewVersion(s)
	return err == nil
}

func isConstraint(s string) bool {
	_, err := semver.NewConstraint(s)
	return err == nil
}

func SaveResolvedNodes(lf *lockfile.Lockfile, nodes map[string]*ResolvedNode, pkg *workspace.Package, updateAliases []string) {
	w, _ := lf.Workspaces.Get(pkg.Rel)
	u := make(map[string]struct{}, len(updateAliases))
	for _, alias := range updateAliases {
		u[alias] = struct{}{}
	}

	for alias, node := range nodes {
		w.Resolutions.Set(alias, node.Resolution)

		// Only format version for specified aliases
		if _, ok := u[alias]; ok {
			finalVersion := node.Specifier.String()
			mapPtr := pkg.Manifest.DependenciesMapPtr(node.Resolution.Mode)
			originalVersion, ok := (*mapPtr)[alias]
			if ok {
				if originalSpec, err := specifier.Parse(fmt.Sprintf("%s@%s", alias, originalVersion)); err == nil {
					finalVersion = formatVersion(originalSpec, node.Specifier)
				}
			}
			if originalVersion != finalVersion {
				pkg.SetDependency(alias, finalVersion, node.Resolution.Mode)
			}
		}
	}

	// Write all package entries to lockfile
	for _, node := range nodes {
		saveNodeToLockfile(lf, node)
	}
}

func saveNodeToLockfile(i *lockfile.Lockfile, node *ResolvedNode) {
	if node.Specifier.Type() == specifier.TypeWorkspace {
		return
	}

	key := node.Specifier.String()
	i.Packages.Take(key, func(old *lockfile.Package, ok bool) (*lockfile.Package, bool) {
		pkg := lockfile.NewPackage(node.Manifest)

		// Collect all resolutions (dependencies + satisfied peers)
		allRes := make(map[string]lockfile.Resolution, len(node.Dependencies)+len(node.SatisfiedPeers))
		for alias, dep := range node.Dependencies {
			allRes[alias] = dep.Resolution
		}
		for peerName, peerResolution := range node.SatisfiedPeers {
			if _, exists := allRes[peerName]; !exists {
				allRes[peerName] = peerResolution
			}
		}

		for alias, resolution := range allRes {
			pkg.Resolutions.Set(alias, resolution)
		}

		return pkg, false
	})

	// Recurse into dependencies
	for _, dep := range node.Dependencies {
		saveNodeToLockfile(i, dep)
	}
}

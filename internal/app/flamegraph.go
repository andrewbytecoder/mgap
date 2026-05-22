package app

import (
	"context"
	"fmt"

	"github.com/andrewbytecoder/mgap/internal/metrics"
	pprofprofile "github.com/moderato-app/pprof/profile"
)

type FlamegraphNode struct {
	Name      string           `json:"name"`
	FullName  string           `json:"fullName"`
	FileName  string           `json:"fileName"`
	Value     int64            `json:"value"`
	SelfValue int64            `json:"selfValue"`
	Children  []FlamegraphNode `json:"children"`
}

type mutableFlamegraphNode struct {
	Name      string
	FullName  string
	FileName  string
	Value     int64
	SelfValue int64
	Children  map[string]*mutableFlamegraphNode
}

func (s *Service) GetProfileFlamegraph(parent context.Context, input string, profile string, seconds uint64) (*FlamegraphNode, error) {
	baseURL, err := normalizeURL(input)
	if err != nil {
		return nil, err
	}

	target, err := profileURL(baseURL, profile, 0, seconds)
	if err != nil {
		return nil, err
	}

	ctx := parent
	if ctx == nil {
		ctx = context.Background()
	}

	data, err := metrics.FetchRawProfile(ctx, target)
	if err != nil {
		return nil, err
	}

	return buildFlamegraphTree(data)
}

func buildFlamegraphTree(data []byte) (*FlamegraphNode, error) {
	prof, err := pprofprofile.ParseData(data)
	if err != nil {
		return nil, err
	}
	if len(prof.Sample) == 0 {
		return nil, fmt.Errorf("profile contains no samples")
	}

	index, err := prof.SampleIndexByName(prof.DefaultSampleType)
	if err != nil {
		return nil, err
	}

	root := &mutableFlamegraphNode{
		Name:     "root",
		FullName: "root",
		Children: map[string]*mutableFlamegraphNode{},
	}

	for _, sample := range prof.Sample {
		if len(sample.Value) <= index {
			continue
		}
		value := sample.Value[index]
		if value <= 0 {
			continue
		}

		current := root
		current.Value += value
		if len(sample.Location) == 0 {
			current.SelfValue += value
		}

		for i := len(sample.Location) - 1; i >= 0; i-- {
			loc := sample.Location[i]
			if loc == nil {
				continue
			}

			if len(loc.Line) == 0 {
				key := fmt.Sprintf("0x%x", loc.Address)
				child, ok := current.Children[key]
				if !ok {
					child = &mutableFlamegraphNode{
						Name:     key,
						FullName: key,
						Children: map[string]*mutableFlamegraphNode{},
					}
					current.Children[key] = child
				}
				child.Value += value
				current = child
				continue
			}

			for j := len(loc.Line) - 1; j >= 0; j-- {
				line := loc.Line[j]
				funcName := "unknown"
				fileName := ""
				if line.Function != nil {
					if line.Function.Name != "" {
						funcName = line.Function.Name
					}
					fileName = line.Function.Filename
				}
				key := fmt.Sprintf("%s|%s|%d", funcName, fileName, line.Line)
				child, ok := current.Children[key]
				if !ok {
					child = &mutableFlamegraphNode{
						Name:     shortName(funcName),
						FullName: funcName,
						FileName: fileName,
						Children: map[string]*mutableFlamegraphNode{},
					}
					current.Children[key] = child
				}
				child.Value += value
				if i == 0 && j == 0 {
					child.SelfValue += value
				}
				current = child
			}
		}
	}

	return flattenFlamegraph(root), nil
}

func flattenFlamegraph(node *mutableFlamegraphNode) *FlamegraphNode {
	children := make([]FlamegraphNode, 0, len(node.Children))
	for _, child := range node.Children {
		children = append(children, *flattenFlamegraph(child))
	}

	return &FlamegraphNode{
		Name:      node.Name,
		FullName:  node.FullName,
		FileName:  node.FileName,
		Value:     node.Value,
		SelfValue: node.SelfValue,
		Children:  children,
	}
}

func shortName(full string) string {
	if idx := len(full) - 1; idx >= 0 {
		lastSlash := -1
		lastDot := -1
		for i := len(full) - 1; i >= 0; i-- {
			if full[i] == '/' {
				lastSlash = i
				break
			}
		}
		for i := len(full) - 1; i >= 0; i-- {
			if full[i] == '.' {
				lastDot = i
				break
			}
		}
		if lastDot > lastSlash && lastDot+1 < len(full) {
			return full[lastDot+1:]
		}
	}
	return full
}

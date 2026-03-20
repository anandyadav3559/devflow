package tests

import (
	"reflect"
	"testing"

	"github.com/anandyadav3559/devflow/services"
)

func TestTopoSort(t *testing.T) {
	tests := []struct {
		name     string
		services map[string]services.Service
		want     []string
		wantErr  bool
	}{
		{
			name: "simple linear dependency",
			services: map[string]services.Service{
				"s1": {Command: "echo"},
				"s2": {Command: "echo", DependsOn: []string{"s1"}},
			},
			want:    []string{"s1", "s2"},
			wantErr: false,
		},
		{
			name: "multiple dependencies",
			services: map[string]services.Service{
				"s1": {Command: "echo"},
				"s2": {Command: "echo"},
				"s3": {Command: "echo", DependsOn: []string{"s1", "s2"}},
			},
			// Both [s1, s2, s3] and [s2, s1, s3] are valid since we sort the queue
			want:    []string{"s1", "s2", "s3"},
			wantErr: false,
		},
		{
			name: "circular dependency",
			services: map[string]services.Service{
				"s1": {Command: "echo", DependsOn: []string{"s2"}},
				"s2": {Command: "echo", DependsOn: []string{"s1"}},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "unknown dependency",
			services: map[string]services.Service{
				"s1": {Command: "echo", DependsOn: []string{"unknown"}},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := services.TopoSort(tt.services)
			if (err != nil) != tt.wantErr {
				t.Errorf("topoSort() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("topoSort() got = %v, want %v", got, tt.want)
			}
		})
	}
}

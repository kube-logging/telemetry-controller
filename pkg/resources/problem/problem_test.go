// Copyright © 2025 Kube logging authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package problem

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type MockProblemHandler struct {
	problems []string
}

func (m *MockProblemHandler) GetProblems() []string {
	return m.problems
}

func (m *MockProblemHandler) SetProblems(problems []string) {
	m.problems = problems
}

func (m *MockProblemHandler) AddProblem(probs ...string) {
	Add(m, probs...)
}

func (m *MockProblemHandler) ClearProblems() {
	m.problems = []string{}
}

func TestAdd(t *testing.T) {
	tests := []struct {
		name             string
		initialProblems  []string
		problemsToAdd    []string
		expectedProblems []string
	}{
		{
			name:             "add single problem to empty list",
			initialProblems:  []string{},
			problemsToAdd:    []string{"problem1"},
			expectedProblems: []string{"problem1"},
		},
		{
			name:             "add multiple problems to empty list",
			initialProblems:  []string{},
			problemsToAdd:    []string{"problem1", "problem2", "problem3"},
			expectedProblems: []string{"problem1", "problem2", "problem3"},
		},
		{
			name:             "add problem to existing list",
			initialProblems:  []string{"existing"},
			problemsToAdd:    []string{"new"},
			expectedProblems: []string{"existing", "new"},
		},
		{
			name:             "add duplicate problem (should not duplicate)",
			initialProblems:  []string{"problem1"},
			problemsToAdd:    []string{"problem1"},
			expectedProblems: []string{"problem1"},
		},
		{
			name:             "add mix of new and duplicate problems",
			initialProblems:  []string{"problem1", "problem2"},
			problemsToAdd:    []string{"problem2", "problem3", "problem1", "problem4"},
			expectedProblems: []string{"problem1", "problem2", "problem3", "problem4"},
		},
		{
			name:             "add empty string (should be ignored)",
			initialProblems:  []string{"problem1"},
			problemsToAdd:    []string{""},
			expectedProblems: []string{"problem1"},
		},
		{
			name:             "add whitespace-only string (should be ignored)",
			initialProblems:  []string{"problem1"},
			problemsToAdd:    []string{"   "},
			expectedProblems: []string{"problem1"},
		},
		{
			name:             "add string with leading/trailing whitespace (should be trimmed)",
			initialProblems:  []string{},
			problemsToAdd:    []string{"  problem1  "},
			expectedProblems: []string{"problem1"},
		},
		{
			name:             "add mix of valid and invalid problems",
			initialProblems:  []string{"existing"},
			problemsToAdd:    []string{"", "  valid  ", "   ", "another"},
			expectedProblems: []string{"existing", "valid", "another"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &MockProblemHandler{
				problems: make([]string, len(tt.initialProblems)),
			}
			copy(handler.problems, tt.initialProblems)

			Add(handler, tt.problemsToAdd...)

			assert.Equal(t, tt.expectedProblems, handler.GetProblems())
		})
	}
}

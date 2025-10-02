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
	"slices"
	"strings"
)

type ProblemHandler interface {
	GetProblems() []string
	SetProblems(problems []string)
	AddProblem(probs ...string)
	ClearProblems()
}

func Add[component ProblemHandler](c component, problems ...string) {
	for _, problem := range problems {
		problem = strings.TrimSpace(problem)
		if problem == "" {
			continue
		}

		if !slices.Contains(c.GetProblems(), problem) {
			c.SetProblems(append(c.GetProblems(), problem))
		}
	}
}

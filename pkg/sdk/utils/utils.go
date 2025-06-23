// Copyright © 2024 Kube logging authors
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

package utils

import (
	"fmt"
	"slices"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ToPtr[T any](v T) *T {
	return &v
}

func ToValue[T any](v *T) T {
	return *v
}

// GetConcreteTypeFromList converts a slice of client.Object to a slice of concrete types.
// Returns an error if any conversion fails.
// Example usage:
//
//	subscriptions, err := GetConcreteTypeFromList[*v1alpha1.Subscription](objectList)
//	outputs, err := GetConcreteTypeFromList[*v1alpha1.Output](objectList)
func GetConcreteTypeFromList[T client.Object](objects []client.Object) ([]T, error) {
	result := make([]T, 0, len(objects))
	for i, obj := range objects {
		if concrete, ok := GetConcreteType[T](obj); !ok {
			return nil, fmt.Errorf("failed to convert object at index %d to type %T", i, *new(T))
		} else {
			result = append(result, concrete)
		}
	}
	return result, nil
}

// GetConcreteType returns the concrete type T from a client.Object if possible.
// Returns the concrete type and true if successful, zero value and false if not.
// Example usage:
//
//	if sub, ok := GetConcreteType[*v1alpha1.Subscription](obj); ok {
//	    // Use sub.Status, sub.Spec etc
//	}
//	if out, ok := GetConcreteType[*v1alpha1.Output](obj); ok {
//	    // Use out.Status, out.Spec etc
//	}
func GetConcreteType[T client.Object](obj client.Object) (T, bool) {
	var zero T
	concrete, ok := obj.(T)
	if !ok {
		return zero, false
	}
	return concrete, true
}

// ToObject converts a slice of concrete types to a slice of client.Object.
func ToObject[T client.Object](items []T) []client.Object {
	objects := make([]client.Object, len(items))
	for i, item := range items {
		objects[i] = item
	}
	return objects
}

// NormalizeStringSlice takes a slice of strings, removes duplicates, sorts it, and returns the unique sorted slice.
func NormalizeStringSlice(inputList []string) []string {
	allKeys := make(map[string]bool)
	uniqueList := []string{}
	for _, item := range inputList {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			uniqueList = append(uniqueList, item)
		}
	}
	slices.Sort(uniqueList)

	return uniqueList
}

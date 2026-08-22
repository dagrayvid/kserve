/*
Copyright 2026 The KServe Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package llmisvc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kserve/kserve/pkg/apis/serving/v1alpha2"
)

func TestHasVLLMArg(t *testing.T) {
	t.Parallel()

	t.Run("found in VLLM_ADDITIONAL_ARGS env", func(t *testing.T) {
		c := &corev1.Container{
			Env: []corev1.EnvVar{{Name: "VLLM_ADDITIONAL_ARGS", Value: "--root-path /foo"}},
		}
		assert.True(t, hasVLLMArg(c, "--root-path"))
	})

	t.Run("found in container args", func(t *testing.T) {
		c := &corev1.Container{
			Args: []string{"--root-path", "/foo"},
		}
		assert.True(t, hasVLLMArg(c, "--root-path"))
	})

	t.Run("found in container command", func(t *testing.T) {
		c := &corev1.Container{
			Command: []string{"vllm", "serve", "--root-path=/foo"},
		}
		assert.True(t, hasVLLMArg(c, "--root-path"))
	})

	t.Run("not present", func(t *testing.T) {
		c := &corev1.Container{
			Env:  []corev1.EnvVar{{Name: "VLLM_ADDITIONAL_ARGS", Value: "--enable-lora"}},
			Args: []string{"--max-model-len", "4096"},
		}
		assert.False(t, hasVLLMArg(c, "--root-path"))
	})

	t.Run("different env var ignored", func(t *testing.T) {
		c := &corev1.Container{
			Env: []corev1.EnvVar{{Name: "OTHER_VAR", Value: "--root-path /foo"}},
		}
		assert.False(t, hasVLLMArg(c, "--root-path"))
	})
}

func TestInjectRootPath(t *testing.T) {
	t.Parallel()

	newLLMSvc := func(ns, name string) *v1alpha2.LLMInferenceService {
		return &v1alpha2.LLMInferenceService{
			ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name},
		}
	}

	t.Run("injects root-path", func(t *testing.T) {
		podSpec := &corev1.PodSpec{
			Containers: []corev1.Container{{Name: "main", Args: []string{"--max-model-len", "4096"}}},
		}
		injectRootPath(newLLMSvc("my-ns", "my-svc"), podSpec)
		assert.Equal(t, []string{"--max-model-len", "4096", "--root-path", "/my-ns/my-svc"}, podSpec.Containers[0].Args)
	})

	t.Run("skips when --root-path already in args", func(t *testing.T) {
		podSpec := &corev1.PodSpec{
			Containers: []corev1.Container{{Name: "main", Args: []string{"--root-path", "/custom"}}},
		}
		injectRootPath(newLLMSvc("my-ns", "my-svc"), podSpec)
		assert.Equal(t, []string{"--root-path", "/custom"}, podSpec.Containers[0].Args)
	})

	t.Run("skips when --root_path (underscore) already set", func(t *testing.T) {
		podSpec := &corev1.PodSpec{
			Containers: []corev1.Container{{Name: "main", Args: []string{"--root_path", "/custom"}}},
		}
		injectRootPath(newLLMSvc("my-ns", "my-svc"), podSpec)
		assert.Equal(t, []string{"--root_path", "/custom"}, podSpec.Containers[0].Args)
	})

	t.Run("skips when --root-path in VLLM_ADDITIONAL_ARGS", func(t *testing.T) {
		podSpec := &corev1.PodSpec{
			Containers: []corev1.Container{{
				Name: "main",
				Env:  []corev1.EnvVar{{Name: "VLLM_ADDITIONAL_ARGS", Value: "--root-path /custom"}},
			}},
		}
		injectRootPath(newLLMSvc("my-ns", "my-svc"), podSpec)
		assert.Empty(t, podSpec.Containers[0].Args)
	})

	t.Run("no-op when main container missing", func(t *testing.T) {
		podSpec := &corev1.PodSpec{
			Containers: []corev1.Container{{Name: "sidecar"}},
		}
		injectRootPath(newLLMSvc("my-ns", "my-svc"), podSpec)
		assert.Empty(t, podSpec.Containers[0].Args)
	})
}

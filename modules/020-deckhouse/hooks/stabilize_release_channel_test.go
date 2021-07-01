/*
Copyright 2021 Flant CJSC

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

package hooks

import (
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"github.com/deckhouse/deckhouse/go_lib/dependency"
	. "github.com/deckhouse/deckhouse/testing/hooks"
)

var _ = Describe("Modules :: deckhouse :: hooks :: stabilize release channel ::", func() {
	const (
		releaseChannelKey = "deckhouse.releaseChannel"
		imageKey          = "deckhouse.internal.currentReleaseImageName"
	)

	Context("no desired release channel", func() {
		f := HookExecutionConfigInit(` {
			"global": {
				"deckhouseVersion": "12345",
				"modulesImages": {
					"registry": "registry.flant.com/sys/deckhouse-oss"
				}
			},
			"deckhouse": {
				"internal": {
					"currentReleaseImageName": "test"
				}
			}
		}`, `{}`)

		BeforeEach(func() {
			f.RunHook()
		})

		It("Hook does not fail", func() {
			Expect(f).Should(ExecuteSuccessfully())
		})

		It("Hook does not change values", func() {
			tag := f.ValuesGet(imageKey).String()
			Expect(tag).To(Equal("test"))
		})
	})

	Context("current image not from release channel", func() {
		f := HookExecutionConfigInit(` {
			"global": {
				"deckhouseVersion": "12345",
				"modulesImages": {
					"registry": "registry.flant.com/sys/deckhouse-oss"
				}
			},
			"deckhouse": {
				"releaseChannel": "EarlyAccess",
				"internal": {
					"currentReleaseImageName": "test"
				}
			}
		}`, `{}`)

		BeforeEach(func() {
			f.RunHook()
		})

		It("Hook does not fail", func() {
			Expect(f).Should(ExecuteSuccessfully())
		})

		It("Hook does not change values", func() {
			tag := f.ValuesGet(imageKey).String()
			Expect(tag).To(Equal("test"))
		})
	})

	Context("current image is from the desired release channel", func() {
		f := HookExecutionConfigInit(` {
			"global": {
				"deckhouseVersion": "12345",
				"modulesImages": {
					"registry": "registry.flant.com/sys/deckhouse-oss"
				}
			},
			"deckhouse": {
				"releaseChannel": "EarlyAccess",
				"internal": {
					"currentReleaseImageName": "registry.flant.com/sys/deckhouse-oss:early_access"
				}
			}
		}`, `{}`)

		table.DescribeTable("tags", func(rcName, rcTag string) {
			f.ValuesSet(releaseChannelKey, rcName)
			oldImage := "registry.flant.com/sys/deckhouse-oss:" + rcTag
			f.ValuesSet(imageKey, oldImage)

			f.RunHook()

			Expect(f).Should(ExecuteSuccessfully())
			newImage := f.ValuesGet(imageKey).String()
			Expect(newImage).To(Equal(oldImage))
		},
			table.Entry("Alpha", nameAlpha, tagAlpha),
			table.Entry("Beta", nameBeta, tagBeta),
			table.Entry("EarlyAccess", nameEarlyAccess, tagEarlyAccess),
			table.Entry("Stable", nameStable, tagStable),
			table.Entry("RockSolid", nameRockSolid, tagRockSolid),
		)
	})

	Context("upgrading release", func() {
		f := HookExecutionConfigInit(` {
			"global": {
				"deckhouseVersion": "12345",
				"modulesImages": {
					"registry": "registry.flant.com/sys/deckhouse-oss"
				}
			},
			"deckhouse": {
				"releaseChannel": "EarlyAccess",
				"internal": {
					"currentReleaseImageName": "registry.flant.com/sys/deckhouse-oss:early_access"
				}
			}
		}`, `{}`)

		mockDigestPerReleaseChannel := func() {
			dependency.TestDC.CRClient.DigestMock.Set(func(tag string) (string, error) {
				return tag, nil
			})
		}

		table.DescribeTable("tags upgrade by one for different digests", func(currentRelease, desiredRelease string) {
			mockDigestPerReleaseChannel()

			currentReleaseChannel := releaseChannelFromName(currentRelease)
			expectedReleaseChannel := currentReleaseChannel - 1

			currentImage := "registry.flant.com/sys/deckhouse-oss:" + currentReleaseChannel.Tag()
			expectedImage := "registry.flant.com/sys/deckhouse-oss:" + expectedReleaseChannel.Tag()

			f.ValuesSet(releaseChannelKey, desiredRelease)
			f.ValuesSet(imageKey, currentImage)

			f.RunHook()

			Expect(f).Should(ExecuteSuccessfully())

			resultImage := f.ValuesGet(imageKey).String()
			Expect(resultImage).To(Equal(expectedImage))
		},

			// upgrade by one
			table.Entry("Beta -> Alpha", nameBeta, nameAlpha),
			table.Entry("EarlyAccess -> Beta", nameEarlyAccess, nameBeta),
			table.Entry("Stable -> EarlyAccess", nameStable, nameEarlyAccess),
			table.Entry("RockSolid -> Stable", nameRockSolid, nameStable),

			// upgrade by two
			table.Entry("EarlyAccess -> Alpha", nameEarlyAccess, nameAlpha),
			table.Entry("Stable -> Beta", nameStable, nameBeta),
			table.Entry("RockSolid -> EarlyAccess", nameRockSolid, nameEarlyAccess),
		)

		table.DescribeTable("tags don't downgrade for different digests", func(currentRelease, desiredRelease string) {
			mockDigestPerReleaseChannel()

			currentReleaseChannel := releaseChannelFromName(currentRelease)

			currentImage := "registry.flant.com/sys/deckhouse-oss:" + currentReleaseChannel.Tag()

			f.ValuesSet(releaseChannelKey, desiredRelease)
			f.ValuesSet(imageKey, currentImage)

			f.RunHook()

			Expect(f).Should(ExecuteSuccessfully())

			resultImage := f.ValuesGet(imageKey).String()
			Expect(resultImage).To(Equal(currentImage))
		},
			// downgrade by one
			table.Entry("Alpha -> Beta", nameAlpha, nameBeta),
			table.Entry("Beta -> EarlyAccess", nameBeta, nameEarlyAccess),
			table.Entry("EarlyAccess -> Stable", nameEarlyAccess, nameStable),
			table.Entry("Stable -> RockSolid", nameStable, nameRockSolid),

			// downgrade by two
			table.Entry("Alpha -> EarlyAccess", nameAlpha, nameEarlyAccess),
			table.Entry("Beta -> Stable", nameBeta, nameStable),
			table.Entry("EarlyAccess -> RockSolid", nameEarlyAccess, nameRockSolid),
		)
	})

	Context("shifting release channel to first different digest", func() {
		f := HookExecutionConfigInit(` {
			"global": {
				"deckhouseVersion": "12345",
				"modulesImages": {
					"registry": "registry.flant.com/sys/deckhouse-oss"
				}
			},
			"deckhouse": {
				"releaseChannel": "EarlyAccess",
				"internal": {
					"currentReleaseImageName": "registry.flant.com/sys/deckhouse-oss:early_access"
				}
			}
		}`, `{}`)

		mockDistinctDigest := func(different string) {
			dependency.TestDC.CRClient.DigestMock.Set(func(tag string) (string, error) {
				if tag == different {
					return "different", nil
				}
				return "same", nil
			})
		}

		table.DescribeTable("upgrade release channel to first different digest, skipping the same", func(currentRelease, desiredRelease string) {
			currentTag := releaseChannelFromName(currentRelease).Tag()
			desiredTag := releaseChannelFromName(desiredRelease).Tag()

			mockDistinctDigest(desiredTag)

			currentImage := "registry.flant.com/sys/deckhouse-oss:" + currentTag
			desiredImage := "registry.flant.com/sys/deckhouse-oss:" + desiredTag

			f.ValuesSet(releaseChannelKey, desiredRelease)
			f.ValuesSet(imageKey, currentImage)

			f.RunHook()

			Expect(f).Should(ExecuteSuccessfully())
			resultImage := f.ValuesGet(imageKey).String()
			Expect(resultImage).To(Equal(desiredImage))
		},
			// upgrade
			table.Entry("EarlyAccess -> Alpha", nameEarlyAccess, nameAlpha),
			table.Entry("Stable -> Alpha", nameStable, nameAlpha),
			table.Entry("RockSolid -> Alpha", nameRockSolid, nameAlpha),
		)
	})

	Context("parsing current release channel", func() {
		const (
			repo             = "registry.flant.com/sys/deckhouse-oss"
			imageFromRepo    = "registry.flant.com/sys/deckhouse-oss:early-access"
			imageNotFromRepo = "registry.flant.com/experiments:early-access"
		)

		It("parses known release channel from the repo", func() {
			rc, isKnown := getCurrentChannel(imageFromRepo, repo)

			Expect(rc).To(Equal(earlyAccessReleaseChannel))
			Expect(isKnown).To(BeTrue())
		})

		It("parses invalid release channel from the repo", func() {
			rc, isKnown := getCurrentChannel(imageNotFromRepo, repo)

			Expect(rc).To(Equal(unknownReleaseChannel))
			Expect(isKnown).To(BeFalse())
		})
	})
})

// Copyright 2023 Specter Ops, Inc.
//
// Licensed under the Apache License, Version 2.0
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

import JSZip from "jszip";


export const FILE_TYPES = { // is this something that is going to be worth it?? or should we skip?
	zip: {
		ext: 'zip',
		mime: 'application/zip'
	},
	json: {
		ext: 'json',
		mime: 'application/json'
	}
} as const

type acceptedMimeTypes = keyof typeof FILE_TYPES

const MAX_FILE_SIZE = 1000000000;
const DEFAULT_ACCEPTED_MIME_TYPES: Array<acceptedMimeTypes> = ['json', 'zip'];

/**
 * 
 * @param file File to be validated
 * @param acceptedTypes files types that are allowed. Defaults to application/json and application/zip
 * @returns 
 */
export const validateFile = async (file: File, acceptedTypes = DEFAULT_ACCEPTED_MIME_TYPES) => {
	const acceptedMap = acceptedTypes.reduce((a: Record<string, boolean>, c) => {
		a[FILE_TYPES[c].ext] = true
		a[FILE_TYPES[c].ext] = true
		return a
	}, {})
	const errors = [];

	const invalidFileType = acceptedMap[file.type]
	if (invalidFileType) {
		errors.push('Invalid file type');
	}

	if (file.type === 'application/zip') {
		const jsZip = new JSZip()
		const zipFiles = await openZip(jsZip, file)

		if (!zipFiles) {
			errors.push('Zip is empty')
			return errors
		};


		console.log('zipFiles -- COPY OBJECT AND SEND TO BEN PLEASE :) ', zipFiles) // TODO: remove after testing

		const zipContainsInvalidTypes = Object.entries(zipFiles)
			.some(([f, m]) => {
				// get the EXACT string from a mac, windows, and linux machine - referring to above
				const metaDataFiles = f.startsWith('__')
				if (metaDataFiles) return false

				const filenameParts = f.split('.')
				if (filenameParts.length > 1) {
					const ext = filenameParts.pop()
					if (!ext) return false

					const invalidType = !acceptedMap[ext]
					return invalidType
				}

				return false
			})

		if (zipContainsInvalidTypes) {
			errors.push('Zip contains invalid file types')
		}
	}

	if (file.type === 'application/json' && file.size > MAX_FILE_SIZE) {
		errors.push('File cannot be larger than 1 GB');
	}

	return errors
}

const openZip = async (jsZip: JSZip, file: File) => {
	const zipData = await jsZip.loadAsync(file);
	if (zipData.files) {
		return zipData.files
	}
}

export const formatAcceptedTypes = (accepted: string | Array<string>) => {
	const formatAsExtension = (t: string) => {
		if (t.startsWith('.') || t.includes('/')) {
			return t
		}
		return `.${t}`
	}

	if (typeof accepted === 'string') {
		return accepted
			.split(',')
			.map(formatAsExtension)
	}

	return accepted.map(formatAsExtension)
}

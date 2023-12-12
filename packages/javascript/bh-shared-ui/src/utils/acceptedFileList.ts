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

const MAX_FILE_SIZE = 1000000000;
const ACCEPTED_MIME_TYPES = ['application/json', 'application/zip'];

export const validateFile = async (file: File) => {
	const errors = [];

	if (!ACCEPTED_MIME_TYPES.includes(file.type)) {
		errors.push('Invalid file type');
	}

	if (file.type === 'application/zip') {
		const jsZip = new JSZip()
		const zipFiles = await openZip(jsZip, file)

		if (!zipFiles) {
			errors.push('Zip is empty')
			return
		};
		console.log('zipFiles', zipFiles) // remove after testing
		const containsNonJsonFiles = Object.entries(zipFiles)
			.some(async ([f, m]) => {
				const notJson = !f.toLowerCase().endsWith('.json')
				const notMetaDataFile = !f.startsWith('__') // get the EXACT string from a mac, windows, and linux machine 

				return notMetaDataFile && notJson
			})

		if (containsNonJsonFiles) errors.push('Zip contains non JSON files')
	} else if (file.size > MAX_FILE_SIZE) {
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
		if (t.startsWith('.')) {
			return t
		}
		// if in dev maybe print a hint
		return `.${t}`
	}

	if (typeof accepted === 'string') {
		return accepted
			.split(',')
			.map(formatAsExtension)
	}

	return accepted.map(formatAsExtension)
}

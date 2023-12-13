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

const MAX_FILE_SIZE = 1000000000;
const DEFAULT_ACCEPTED_MIME_TYPES: Array<keyof typeof FILE_TYPES> = ['json', 'zip']; // should this be mime or ext?

/**
 * 
 * @param file File to be validated
 * @param acceptedTypes files types that are allowed. Defaults to application/json and application/zip
 * @returns 
 */
export const validateFile = async (file: File, acceptedTypes?: Array<keyof typeof FILE_TYPES>) => {
	const accepted = acceptedTypes ?? DEFAULT_ACCEPTED_MIME_TYPES
	const errors = [];


	const invalidFileType = accepted.every(type => FILE_TYPES[type].mime !== file.type)
	if (invalidFileType) {
		errors.push('Invalid file type');
	}

	if (file.type === 'application/zip') { // is there another way for these to show up
		const jsZip = new JSZip()
		const zipFiles = await openZip(jsZip, file)

		if (!zipFiles) {
			errors.push('Zip is empty')
			return errors
		};


		console.log('zipFiles -- COPY OBJECT AND SEND TO BEN PLEASE :) ', zipFiles) // remove after testing
		const containsNonJsonFiles = Object.entries(zipFiles)
			.some(async ([f, m]) => {
				const notJson = !f.toLowerCase().endsWith('.json') // check that it ends with an acceptedType
				const notMetaDataFile = !f.startsWith('__') // get the EXACT string from a mac, windows, and linux machine 

				return notMetaDataFile && notJson
			})

		if (containsNonJsonFiles) {
			errors.push('Zip contains non JSON files')
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

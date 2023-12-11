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

export const acceptedFileList = async (fileList: FileList, acceptedTypes: string | Array<string>) => {
	let _acceptedTypes = acceptedTypes
	if (!_acceptedTypes) return;
	if (typeof _acceptedTypes === 'string') _acceptedTypes = _acceptedTypes.split(',')

	const isAccepted = true;
	const jsZip = new JSZip()
	for (let i = 0; i < fileList.length; i++) {
		const file = fileList[i]
		if (file.type === 'application/zip') {
			const zipFiles = await openZip(jsZip, file)
			console.log('zipFiles', zipFiles)
			if (!zipFiles) return; // what do we do when zip file is empty? return false probably
			Object.entries(zipFiles).every(([fileName, meta]) => {
				console.log('name', fileName)
				return validateZippedFile(meta)
			})
		} else {
			validateFile(file, _acceptedTypes)
		}
	}
	return isAccepted
}

const validateFile = (file: File, allowedTypes: Array<string>) => {
	console.log('file', file)
}

const validateZippedFile = (fileMeta: JSZip.JSZipObject) => {
	console.log('fileMeta', fileMeta)
}


const openZip = async (jsZip: JSZip, file: File) => {
	const zipData = await jsZip.loadAsync(file);
	if (zipData.files) {
		return zipData.files
	}
}

// const getFileMagicNumber = () => {
// 	// we can check if the magic number is something else other than text?
// }
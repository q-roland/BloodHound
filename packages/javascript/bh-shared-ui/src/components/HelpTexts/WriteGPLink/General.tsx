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

import { FC } from 'react';
import { groupSpecialFormat } from '../utils';
import { EdgeInfoProps } from '../index';
import { Typography } from '@mui/material';

const General: FC<EdgeInfoProps> = ({ sourceName, sourceType, targetName }) => {
    return (
        <>
            <Typography variant='body2'>
                {groupSpecialFormat(sourceType, sourceName)} has the permissions to modify the gPLink attribute of the 
                Organizational Unit {targetName}.
            </Typography>

            <Typography variant='body2'>
                The ability to alter the gPLink attribute of an Organizational Unit may allow an attacker to apply a malicious 
                Group Policy Object to all of the OU's child items (including the ones located in nested sub-OUs). This can be exploited 
                to make said child items execute arbitrary commands through an immediate scheduled task, thus compromising them.
            </Typography>

            <Typography variant='body2'>
                Successful exploitation will require the possibility to add non-existing DNS records to the domain and to 
                create machine accounts. Alternatively, an already compromised domain-joined machine may be used to perform the attack.
                Note that the attack vector implementation is not trivial and will require some setup.
            </Typography>

            <Typography variant='body2'>
                It can also be mentioned that the ability to modify the gPLink attribute of an Organizational Unit can be exploited in 
                conjunction with write permissions on a GPO. Indeed, in such a situation, an attacker could first inject a malicious scheduled 
                task in the controlled GPO, and then link the GPO to the target OU through its gPLink attribute, making all child objects apply the 
                malicious GPO and execute arbitrary commands.
            </Typography>
        </>
    );
};

export default General;
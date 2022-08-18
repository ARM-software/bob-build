/*
 * Copyright 2022 Arm Limited.
 * SPDX-License-Identifier: Apache-2.0
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

void output_single();
void output_multiple_in();
void output_multiple_out();
void output_multiple_out2();
void output_multiple_in_out();
void output_multiple_in_out2();

void output_level_1_single();
void output_level_2_single();
void output_level_3_single();
void output_extra_single();

void output_deps();
void output_deps2();

void validate_link()
{
	output_single();
	output_multiple_in();
	output_multiple_out();
	output_multiple_out2();
	output_multiple_in_out();
	output_multiple_in_out2();

	output_level_1_single();
	output_level_2_single();
	output_level_3_single();
	output_extra_single();

	output_deps();
	output_deps2();
}

int main()
{
	validate_link();
}

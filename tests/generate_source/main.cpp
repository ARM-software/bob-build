void output_single();
void output_not_unique_in();
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
	output_not_unique_in();
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

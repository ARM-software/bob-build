#include <iostream>
#include <fstream>

int main(int argc, char* argv[]) {

    if (argc != 2) {
        std::cerr << "Only one param is expected. Got [" << argc - 1 << "]" << std::endl;
        return EXIT_FAILURE;
    }

    std::ofstream out_file(argv[1]);

    out_file << "// Dummy output: " << argv[1] << std::endl;
    out_file.close();

    return EXIT_SUCCESS;
}

# Contributing to homegit

Thank you for considering contributing to homegit! 🎉

## How to Contribute

### Reporting Bugs

1. Check if the bug has already been reported in [Issues](https://github.com/chris-roerig/homegit/issues)
2. If not, create a new issue using the bug report template
3. Include:
   - Your OS and Go version
   - Steps to reproduce
   - Expected vs actual behavior
   - Relevant logs (use `homegit logs`)

### Suggesting Features

1. Check [Issues](https://github.com/chris-roerig/homegit/issues) for existing feature requests
2. Create a new issue using the feature request template
3. Describe:
   - The problem you're trying to solve
   - Your proposed solution
   - Any alternatives you've considered

### Pull Requests

1. **Fork** the repository
2. **Create a branch** from `main`:
   ```bash
   git checkout -b feature/your-feature-name
   ```
3. **Make your changes**:
   - Follow Go conventions and formatting (`gofmt`)
   - Add tests if applicable
   - Update documentation if needed
4. **Test your changes**:
   ```bash
   go test ./...
   go build -o homegit
   ```
5. **Commit** with clear messages:
   ```bash
   git commit -m "Add feature: description"
   ```
6. **Push** to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```
7. **Open a Pull Request** against `main`

### Code Style

- Follow standard Go conventions
- Run `gofmt` before committing
- Keep functions small and focused
- Add comments for exported functions
- Write clear commit messages

### Testing

- Add tests for new features
- Ensure existing tests pass: `go test ./...`
- Test on your platform before submitting

### Development Setup

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/homegit.git
cd homegit

# Build
go build -o homegit

# Run tests
go test ./...

# Install locally
make install
```

## Questions?

Feel free to open an issue for any questions about contributing!

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

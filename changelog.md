# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- Replaced the legacy method set (backtracking, A-star, ant colony, minimax) with constraint propagation over bitmask candidates, exact cover over the boolean incidence matrix, simulated annealing with a restart portfolio, and a parallel portfolio race.
- Input errors now print to stderr and exit with code 1; the happy-path stdout format is unchanged.

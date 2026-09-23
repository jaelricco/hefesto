// Package skills models the skill graph: families, skills, their ordered
// levels, and the typed edges between levels that form the DAG.
//
// Skills are read by the map. Exercises are written by the logger. The two are
// joined through skill_level_exercises and must never be conflated.
package skills

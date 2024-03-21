package pgsql

//type Ripper struct {
//	CTEs []*CTE
//}
//
//func (s *Ripper) EnterMatch(stack *model.WalkStack, match *model.Match) error {
//	for _, pattern := range match.Pattern {
//		var cte *CTE
//
//		for patternElementIdx, patternElement := range pattern.PatternElements {
//			if nodePattern, isNodePattern := patternElement.AsNodePattern(); isNodePattern {
//				// If this is a back to back node pattern then we need to author a CTE for just this pattern
//				if nextIdx := patternElementIdx + 1; nextIdx < len(pattern.PatternElements) && pattern.PatternElements[nextIdx].IsNodePattern() {
//					cte = &CTE{
//						SyntaxNode: &NodeSelect{
//							Pattern:  nodePattern,
//							Criteria: &CriteriaBuilder{},
//						},
//					}
//
//					s.CTEs = append(s.CTEs, cte)
//				}
//			} else {
//				// Bind the relationship pattern
//				relationshipPattern, _ := patternElement.AsRelationshipPattern()
//
//				// If this is a relationship pattern we need to author a recursive CTE
//				var (
//					rootNodePatternIdx     int
//					terminalNodePatternIdx int
//				)
//
//				switch relationshipPattern.Direction {
//				case graph.DirectionOutbound:
//					rootNodePatternIdx = patternElementIdx - 1
//					terminalNodePatternIdx = patternElementIdx + 1
//
//				case graph.DirectionInbound:
//					rootNodePatternIdx = patternElementIdx + 1
//					terminalNodePatternIdx = patternElementIdx - 1
//
//				default:
//					return fmt.Errorf("unsupported direction")
//				}
//
//				if rootNodePattern, isNodePattern := pattern.PatternElements[rootNodePatternIdx].AsNodePattern(); !isNodePattern {
//					return fmt.Errorf("relationship without root node pattern element")
//				} else if terminalNodePattern, isNodePattern := pattern.PatternElements[terminalNodePatternIdx].AsNodePattern(); !isNodePattern {
//					return fmt.Errorf("relationship without terminal node pattern element")
//				} else {
//					cte = &CTE{
//						Traversal: &Traversal{
//							Direction:            relationshipPattern.Direction,
//							RootNodePattern:      rootNodePattern,
//							EdgePattern:          relationshipPattern,
//							TerminalNodePattern:  terminalNodePattern,
//							RootNodeCriteria:     &CriteriaBuilder{},
//							EdgeCriteria:         &CriteriaBuilder{},
//							TerminalNodeCriteria: &CriteriaBuilder{},
//						},
//					}
//
//					s.CTEs = append(s.CTEs, cte)
//				}
//			}
//		}
//	}
//
//	return nil
//}
//
//func (s *Ripper) Enter(stack *model.WalkStack, expression model.Expression) error {
//	switch typedExpression := expression.(type) {
//	case *model.Match:
//		return s.EnterMatch(stack, typedExpression)
//	}
//
//	return nil
//}
//
//func (s *Ripper) Exit(stack *model.WalkStack, expression model.Expression) error {
//	return nil
//}

package service

import (
	"context"
	"math/rand"

	"github.com/interview-platform/interview-service/internal/domain"

	"github.com/google/uuid"
)

type QuestionGenerationRequest struct {
	InterviewID uuid.UUID
	Difficulty  domain.DifficultyLevel
	Tags        []string
	Count       int
	Duration    int
}

type QuestionGenerator struct {
	questionBank []QuestionTemplate
}

type QuestionTemplate struct {
	Title       string
	Description string
	Type        domain.QuestionType
	Difficulty  domain.DifficultyLevel
	Tags        []string
	StarterCode string
	Solution    string
	TestCases   []domain.TestCase
	Points      int
}

func NewQuestionGenerator() *QuestionGenerator {
	return &QuestionGenerator{
		questionBank: defaultQuestionBank(),
	}
}

func (qg *QuestionGenerator) GenerateQuestions(ctx context.Context, req *QuestionGenerationRequest) ([]*domain.Question, error) {
	if req.Difficulty == "" {
		req.Difficulty = domain.DifficultyMedium
	}

	if req.Count <= 0 {
		req.Count = 3
	}

	// Filter questions by difficulty and tags
	candidates := qg.filterQuestions(req.Difficulty, req.Tags)

	if len(candidates) < req.Count {
		// Fall back to any difficulty if not enough matches
		candidates = qg.questionBank
	}

	// Shuffle and pick
	shuffleQuestions(candidates)

	count := req.Count
	if len(candidates) < count {
		count = len(candidates)
	}

	questions := make([]*domain.Question, 0, count)
	for i := 0; i < count; i++ {
		t := candidates[i]
		questions = append(questions, &domain.Question{
			ID:          uuid.New(),
			InterviewID: req.InterviewID,
			Title:       t.Title,
			Description: t.Description,
			Type:        t.Type,
			Difficulty:  t.Difficulty,
			Tags:        t.Tags,
			StarterCode: t.StarterCode,
			Solution:    t.Solution,
			TestCases:   t.TestCases,
			Points:      t.Points,
			Order:       i + 1,
		})
	}

	return questions, nil
}

func (qg *QuestionGenerator) filterQuestions(difficulty domain.DifficultyLevel, tags []string) []QuestionTemplate {
	var filtered []QuestionTemplate

	for _, q := range qg.questionBank {
		if q.Difficulty != difficulty {
			continue
		}

		if len(tags) > 0 {
			matched := false
			for _, tag := range tags {
				for _, qTag := range q.Tags {
					if qTag == tag {
						matched = true
						break
					}
				}
				if matched {
					break
				}
			}
			if !matched {
				continue
			}
		}

		filtered = append(filtered, q)
	}

	return filtered
}

func shuffleQuestions(questions []QuestionTemplate) {
	r := rand.New(rand.NewSource(42)) //nolint:gosec // seeded for reproducibility
	r.Shuffle(len(questions), func(i, j int) {
		questions[i], questions[j] = questions[j], questions[i]
	})
}

func defaultQuestionBank() []QuestionTemplate {
	return []QuestionTemplate{
		{
			Title:       "Two Sum",
			Description: "Given an array of integers nums and an integer target, return indices of the two numbers such that they add up to target.",
			Type:        domain.QuestionTypeCoding,
			Difficulty:  domain.DifficultyEasy,
			Tags:        []string{"arrays", "hash-table"},
			StarterCode: `func twoSum(nums []int, target int) []int {
    // Write your solution here
    
}`,
			Solution: `func twoSum(nums []int, target int) []int {
    seen := make(map[int]int)
    for i, num := range nums {
        if j, ok := seen[target-num]; ok {
            return []int{j, i}
        }
        seen[num] = i
    }
    return nil
}`,
			TestCases: []domain.TestCase{
				{Input: "[2,7,11,15], 9", Output: "[0,1]", IsHidden: false},
				{Input: "[3,2,4], 6", Output: "[1,2]", IsHidden: false},
				{Input: "[3,3], 6", Output: "[0,1]", IsHidden: true},
			},
			Points: 10,
		},
		{
			Title:       "Valid Parentheses",
			Description: "Given a string s containing just the characters '(', ')', '{', '}', '[' and ']', determine if the input string is valid.",
			Type:        domain.QuestionTypeCoding,
			Difficulty:  domain.DifficultyEasy,
			Tags:        []string{"strings", "stack"},
			StarterCode: `func isValid(s string) bool {
    // Write your solution here
    
}`,
			Solution: `func isValid(s string) bool {
    stack := []rune{}
    pairs := map[rune]rune{')': '(', '}': '{', ']': '['}
    
    for _, c := range s {
        if open, ok := pairs[c]; ok {
            if len(stack) == 0 || stack[len(stack)-1] != open {
                return false
            }
            stack = stack[:len(stack)-1]
        } else {
            stack = append(stack, c)
        }
    }
    return len(stack) == 0
}`,
			TestCases: []domain.TestCase{
				{Input: `"()"`, Output: "true", IsHidden: false},
				{Input: `"()[]{}"`, Output: "true", IsHidden: false},
				{Input: `"(]"`, Output: "false", IsHidden: true},
			},
			Points: 10,
		},
		{
			Title:       "Design a URL Shortener",
			Description: "Design a system that takes a long URL and generates a short URL. The system should also be able to redirect the short URL back to the original URL.",
			Type:        domain.QuestionTypeSystemDesign,
			Difficulty:  domain.DifficultyMedium,
			Tags:        []string{"system-design", "scalability"},
			StarterCode: `// Design the API endpoints and data model
// Consider: encoding scheme, storage, caching, analytics`,
			Solution:  "",
			TestCases: []domain.TestCase{},
			Points:    20,
		},
		{
			Title:       "Binary Search Tree Validation",
			Description: "Given the root of a binary tree, determine if it is a valid binary search tree (BST).",
			Type:        domain.QuestionTypeCoding,
			Difficulty:  domain.DifficultyMedium,
			Tags:        []string{"trees", "recursion"},
			StarterCode: `type TreeNode struct {
    Val   int
    Left  *TreeNode
    Right *TreeNode
}

func isValidBST(root *TreeNode) bool {
    // Write your solution here
    
}`,
			Solution: `func isValidBST(root *TreeNode) bool {
    return validate(root, nil, nil)
}

func validate(node *TreeNode, min, max *int) bool {
    if node == nil {
        return true
    }
    if min != nil && node.Val <= *min {
        return false
    }
    if max != nil && node.Val >= *max {
        return false
    }
    return validate(node.Left, min, &node.Val) && 
           validate(node.Right, &node.Val, max)
}`,
			TestCases: []domain.TestCase{
				{Input: "[2,1,3]", Output: "true", IsHidden: false},
				{Input: "[5,1,4,null,null,3,6]", Output: "false", IsHidden: true},
			},
			Points: 20,
		},
		{
			Title:       "Debug a Race Condition",
			Description: "The following code has a race condition. Identify it and fix it.",
			Type:        domain.QuestionTypeDebugging,
			Difficulty:  domain.DifficultyMedium,
			Tags:        []string{"concurrency", "debugging"},
			StarterCode: `type Counter struct {
    count int
}

func (c *Counter) Increment() {
    c.count++
}

func (c *Counter) Get() int {
    return c.count
}`,
			Solution: `type Counter struct {
    mu    sync.Mutex
    count int
}

func (c *Counter) Increment() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.count++
}

func (c *Counter) Get() int {
    c.mu.Lock()
    defer c.mu.Unlock()
    return c.count
}`,
			TestCases: []domain.TestCase{},
			Points:    25,
		},
		{
			Title:       "Tell me about a time you handled a production incident",
			Description: "Describe a situation where you had to deal with a critical production issue. What was your approach, and what did you learn?",
			Type:        domain.QuestionTypeBehavioral,
			Difficulty:  domain.DifficultyEasy,
			Tags:        []string{"behavioral", "communication"},
			StarterCode: "",
			Solution:    "",
			TestCases:   []domain.TestCase{},
			Points:      15,
		},
		{
			Title:       "Implement LRU Cache",
			Description: "Design and implement a Least Recently Used (LRU) cache that supports get and put operations in O(1) time complexity.",
			Type:        domain.QuestionTypeCoding,
			Difficulty:  domain.DifficultyHard,
			Tags:        []string{"design", "hash-table", "linked-list"},
			StarterCode: `type LRUCache struct {
    capacity int
}

func Constructor(capacity int) LRUCache {
    
}

func (this *LRUCache) Get(key int) int {
    
}

func (this *LRUCache) Put(key int, value int) {
    
}`,
			Solution: `type Node struct {
    key, val   int
    prev, next *Node
}

type LRUCache struct {
    capacity int
    cache    map[int]*Node
    head     *Node
    tail     *Node
}

func Constructor(capacity int) LRUCache {
    head := &Node{}
    tail := &Node{}
    head.next = tail
    tail.prev = head
    return LRUCache{
        capacity: capacity,
        cache:    make(map[int]*Node),
        head:     head,
        tail:     tail,
    }
}

func (this *LRUCache) Get(key int) int {
    if node, ok := this.cache[key]; ok {
        this.moveToHead(node)
        return node.val
    }
    return -1
}

func (this *LRUCache) Put(key int, value int) {
    if node, ok := this.cache[key]; ok {
        node.val = value
        this.moveToHead(node)
        return
    }
    
    node := &Node{key: key, val: value}
    this.cache[key] = node
    this.addToHead(node)
    
    if len(this.cache) > this.capacity {
        removed := this.removeTail()
        delete(this.cache, removed.key)
    }
}`,
			TestCases: []domain.TestCase{
				{Input: `["put", "get", "put", "get"], [[1,1], [1], [2,2], [2]]`, Output: "[null, 1, null, 2]", IsHidden: false},
			},
			Points: 30,
		},
		{
			Title:       "Design a Distributed Message Queue",
			Description: "Design a distributed message queue system that supports publish/subscribe and point-to-point messaging patterns.",
			Type:        domain.QuestionTypeSystemDesign,
			Difficulty:  domain.DifficultyHard,
			Tags:        []string{"system-design", "distributed-systems", "messaging"},
			StarterCode: `// Design considerations:
// - Message ordering guarantees
// - Fault tolerance
// - Consumer groups
// - Message persistence
// - Scaling strategies`,
			Solution:  "",
			TestCases: []domain.TestCase{},
			Points:    35,
		},
	}
}

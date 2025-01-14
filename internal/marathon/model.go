package marathon

import (
	"fmt"
	"time"

	"github.com/Broderick-Westrope/tetrigo/tetris"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/stopwatch"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	matrix     tetris.Matrix
	styles     *Styles
	help       help.Model
	keys       *KeyMap
	currentTet *tetris.Tetrimino
	holdTet    *tetris.Tetrimino
	canHold    bool
	fall       *Fall
	scoring    *tetris.Scoring
	bag        *tetris.Bag
	timer      stopwatch.Model
}

func InitialModel(level uint) *Model {
	m := &Model{
		matrix:  tetris.Matrix{},
		styles:  DefaultStyles(),
		help:    help.New(),
		keys:    DefaultKeyMap(),
		scoring: tetris.NewScoring(level),
		holdTet: &tetris.Tetrimino{
			Cells: [][]bool{
				{false, false, false},
				{false, false, false},
				{false, false, false},
			},
			Value: 0,
		},
		canHold: true,
		timer:   stopwatch.NewWithInterval(time.Millisecond),
	}
	m.bag = tetris.NewBag(len(m.matrix))
	m.fall = defaultFall(level)
	m.currentTet = m.bag.Next()
	err := m.matrix.AddTetrimino(m.currentTet)
	if err != nil {
		panic(fmt.Errorf("failed to add tetrimino to matrix: %w", err))
	}
	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.fall.stopwatch.Init(), m.timer.Init())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
		case key.Matches(msg, m.keys.Left):
			err := m.currentTet.MoveLeft(&m.matrix)
			if err != nil {
				panic(fmt.Errorf("failed to move tetrimino left: %w", err))
			}
		case key.Matches(msg, m.keys.Right):
			err := m.currentTet.MoveRight(&m.matrix)
			if err != nil {
				panic(fmt.Errorf("failed to move tetrimino right: %w", err))
			}
		case key.Matches(msg, m.keys.Clockwise):
			err := m.currentTet.Rotate(&m.matrix, true)
			if err != nil {
				panic(fmt.Errorf("failed to rotate tetrimino clockwise: %w", err))
			}
		case key.Matches(msg, m.keys.CounterClockwise):
			err := m.currentTet.Rotate(&m.matrix, false)
			if err != nil {
				panic(fmt.Errorf("failed to rotate tetrimino counter-clockwise: %w", err))
			}
		case key.Matches(msg, m.keys.HardDrop):
			for {
				finished, err := m.lowerTetrimino()
				if err != nil {
					panic(fmt.Errorf("failed to lower tetrimino (hard drop): %w", err))
				}
				if finished {
					break
				}
			}
		case key.Matches(msg, m.keys.SoftDrop):
			m.fall.toggleSoftDrop()
		case key.Matches(msg, m.keys.Hold):
			err := m.holdTetrimino()
			if err != nil {
				panic(fmt.Errorf("failed to hold tetrimino: %w", err))
			}
		}
	case stopwatch.TickMsg:
		if m.fall.stopwatch.ID() != msg.ID {
			break
		}
		_, err := m.lowerTetrimino()
		if err != nil {
			panic(fmt.Errorf("failed to lower tetrimino (tick): %w", err))
		}
	}

	var cmd tea.Cmd
	var cmds []tea.Cmd

	m.timer, cmd = m.timer.Update(msg)
	cmds = append(cmds, cmd)

	m.fall.stopwatch, cmd = m.fall.stopwatch.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	var output = lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.JoinVertical(lipgloss.Right, m.holdView(), m.informationView()),
		m.matrixView(),
		m.bagView(),
	)

	return output + "\n" + m.help.View(m.keys)
}

func (m *Model) matrixView() string {
	var output string
	for row := (len(m.matrix) - 20); row < len(m.matrix); row++ {
		for col := range m.matrix[row] {
			output += m.renderCell(m.matrix[row][col])
		}
		if row < len(m.matrix)-1 {
			output += "\n"
		}
	}

	var rowIndicator string
	for i := 1; i <= 20; i++ {
		rowIndicator += fmt.Sprintf("%d\n", i)
	}
	return lipgloss.JoinHorizontal(lipgloss.Center, m.styles.Playfield.Render(output), m.styles.RowIndicator.Render(rowIndicator))
}

func (m *Model) informationView() string {
	var output string
	output += fmt.Sprintln("Score: ", m.scoring.Total())
	output += fmt.Sprintln("Level: ", m.scoring.Level())
	output += fmt.Sprintln("Cleared: ", m.scoring.Lines())

	elapsed := m.timer.Elapsed().Seconds()
	minutes := int(elapsed) / 60

	output += "Time: "
	if minutes > 0 {
		seconds := int(elapsed) % 60
		output += fmt.Sprintf("%02d:%02d\n", minutes, seconds)
	} else {
		output += fmt.Sprintf("%06.3f\n", elapsed)
	}

	return m.styles.Information.Render(output)
}

func (m *Model) holdView() string {
	output := "Hold:\n" + m.renderTetrimino(m.holdTet, 1)
	return m.styles.Hold.Render(output)
}

func (m *Model) bagView() string {
	output := "Next:\n"
	for i, t := range m.bag.Elements {
		if i > 5 {
			break
		}
		output += "\n" + m.renderTetrimino(&t, 1)
	}
	return m.styles.Bag.Render(output)
}

func (m *Model) renderTetrimino(t *tetris.Tetrimino, background byte) string {
	var output string
	for row := range t.Cells {
		for col := range t.Cells[row] {
			if t.Cells[row][col] {
				output += m.renderCell(t.Value)
			} else {
				output += m.renderCell(background)
			}
		}
		output += "\n"
	}
	return output
}

func (m *Model) renderCell(cell byte) string {
	switch cell {
	case 0:
		return m.styles.ColIndicator.Render("▕ ")
	case 1:
		return m.styles.TetriminoStyles[cell].Render("  ")
	case 'G':
		return "░░"
	default:
		cellStyle, ok := m.styles.TetriminoStyles[cell]
		if ok {
			return cellStyle.Render("██")
		}
	}
	return "??"
}

func (m *Model) holdTetrimino() error {
	if !m.canHold {
		return nil
	}

	// Swap the current tetrimino with the hold tetrimino
	if m.holdTet.Value == 0 {
		m.holdTet = m.currentTet
		m.currentTet = m.bag.Next()
	} else {
		m.holdTet, m.currentTet = m.currentTet, m.holdTet
	}

	m.matrix.RemoveTetrimino(m.holdTet)

	// Reset the position of the hold tetrimino
	var found bool
	for _, t := range tetris.Tetriminos {
		if t.Value == m.holdTet.Value {
			m.holdTet.Pos = t.Pos
			m.holdTet.Pos.Y += (len(m.matrix) - 20)
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("failed to find tetrimino with value '%v'", m.currentTet.Value)
	}

	// Add the current tetrimino to the matrix
	err := m.matrix.AddTetrimino(m.currentTet)
	if err != nil {
		return fmt.Errorf("failed to add tetrimino to matrix: %w", err)
	}

	m.canHold = false
	return nil
}

func (m *Model) lowerTetrimino() (bool, error) {
	if !m.currentTet.CanMoveDown(m.matrix) {
		action := m.matrix.RemoveCompletedLines(m.currentTet)
		m.scoring.ProcessAction(action)
		m.currentTet = m.bag.Next()
		err := m.matrix.AddTetrimino(m.currentTet)
		if err != nil {
			return false, fmt.Errorf("failed to add tetrimino to matrix: %w", err)
		}
		m.canHold = true
		return true, nil
	}

	err := m.currentTet.MoveDown(&m.matrix)
	if err != nil {
		return false, fmt.Errorf("failed to move tetrimino down: %w", err)
	}

	return false, nil
}

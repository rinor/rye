// builtins.go
package evaldo

import (
	"github.com/refaktor/rye/env"
)

// definiraj frame <builtin nargs arg0 arg1>
// definiraj stack []evalframe
// callbui kreira trenuten frame, nastavi bui nargs in vrne
// while loop pogleda naslednji arg, če je literal nastavi arg in poveča argc če je argc nargs potem pokliče frame in iz stacka potegne naslednjega, če ni potem zalopa
// 									če je builtin potem pusha trenuten frame na stack in kreira novega

func Eyr_CallBuiltin(bi env.Builtin, ps *env.ProgramState, arg0_ env.Object, toLeft bool, stack *env.EyrStack) *env.ProgramState {
	arg0 := bi.Cur0     //env.Object(bi.Cur0)
	var arg1 env.Object // := bi.Cur1
	var arg2 env.Object

	if bi.Argsn > 0 && bi.Cur0 == nil {
		if checkFlagsBi(bi, ps, 0) {
			return ps
		}
		if ps.ErrorFlag || ps.ReturnFlag {
			return ps
		}
		arg0 = stack.Pop(ps)
		if ps.ErrorFlag {
			return ps
		}
		if bi.Argsn == 1 {
			ps.Res = bi.Fn(ps, arg0, nil, nil, nil, nil)
			// stack.Push(ps.Res)
		}
	}
	if bi.Argsn > 1 && bi.Cur1 == nil {
		if checkFlagsBi(bi, ps, 1) {
			return ps
		}
		if ps.ErrorFlag || ps.ReturnFlag {
			return ps
		}

		arg1 = stack.Pop(ps)
		if ps.ErrorFlag {
			return ps
		}
		if bi.Argsn == 2 {
			ps.Res = bi.Fn(ps, arg1, arg0, nil, nil, nil)
			// stack.Push(ps.Res)
		}
	}
	if bi.Argsn > 2 && bi.Cur2 == nil {
		if checkFlagsBi(bi, ps, 0) {
			return ps
		}
		if ps.ErrorFlag || ps.ReturnFlag {
			return ps
		}

		arg2 = stack.Pop(ps)
		if ps.ErrorFlag {
			return ps
		}
		if bi.Argsn == 3 {
			ps.Res = bi.Fn(ps, arg2, arg1, arg0, nil, nil)
			//stack.Push(ps.Res)
		}
	}
	return ps
}

// This is separate from CallFuncitonArgsN so it can manage pulling args directly off of the eyr stack
func Eyr_CallFunction(fn env.Function, es *env.ProgramState, leftVal env.Object, toLeft bool, session *env.RyeCtx, stack *env.EyrStack) *env.ProgramState {
	var fnCtx = DetermineContext(fn, es, session)
	if checkErrorReturnFlag(es) {
		return es
	}

	var arg0 env.Object = nil
	for i := fn.Argsn - 1; i >= 0; i-- {
		var stackElem = stack.Pop(es)
		// TODO: Consider doing check once outside of loop once this version is ready as a correctness comparison point
		if es.ErrorFlag {
			return es
		}
		if arg0 == nil {
			arg0 = stackElem
		}
		fnCtx.Set(fn.Spec.Series.Get(i).(env.Word).Index, stackElem)
	}

	// setup
	psX := env.NewProgramState(fn.Body.Series, es.Idx)
	psX.Ctx = fnCtx
	psX.PCtx = es.PCtx
	psX.Gen = es.Gen

	var result *env.ProgramState
	// es.Ser.SetPos(0)
	if fn.Argsn > 0 {
		result = EvalBlockInj(psX, arg0, arg0 != nil)
	} else {
		result = EvalBlock(psX)
	}
	MaybeDisplayFailureOrError(result, result.Idx)

	if result.ForcedResult != nil {
		es.Res = result.ForcedResult
		result.ForcedResult = nil
	} else {
		es.Res = result.Res
	}
	es.ReturnFlag = false
	return es
}

func Eyr_EvalObject(es *env.ProgramState, object env.Object, leftVal env.Object, toLeft bool, session *env.RyeCtx, stack *env.EyrStack, bakein bool) *env.ProgramState {
	//fmt.Print("EVAL OBJECT")
	switch object.Type() {
	case env.BuiltinType:
		bu := object.(env.Builtin)
		if bakein {
			es.Ser.Put(bu)
		} //es.Ser.SetPos(es.Ser.Pos() - 1)
		if checkFlagsBi(bu, es, 333) {
			return es
		}
		return Eyr_CallBuiltin(bu, es, leftVal, toLeft, stack)
	case env.FunctionType:
		fn := object.(env.Function)
		return Eyr_CallFunction(fn, es, leftVal, toLeft, session, stack)

	default:
		es.Res = object
		return es
	}
}

func Eyr_EvalWord(es *env.ProgramState, word env.Object, leftVal env.Object, toLeft bool, stack *env.EyrStack) *env.ProgramState {
	// LOCAL FIRST
	found, object, ctx := findWordValue(es, word)
	if found {
		es = Eyr_EvalObject(es, object, leftVal, toLeft, ctx, stack, true) //ww0128a *
		stack.Push(es, es.Res)
		return es
	} else {
		es.ErrorFlag = true
		if !es.FailureFlag {
			es.Res = *env.NewError2(5, "Word not found: "+word.Inspect(*es.Idx))
		}
		return es
	}
}

func Eyr_EvalLSetword(ps *env.ProgramState, word env.LSetword, leftVal env.Object, toLeft bool, stack *env.EyrStack) *env.ProgramState {
	idx := word.Index
	val := stack.Pop(ps)
	if ps.ErrorFlag {
		return ps
	}
	ps.Ctx.Mod(idx, val)
	return ps
}

func Eyr_EvalExpression(es *env.ProgramState, stack *env.EyrStack) *env.ProgramState {
	object := es.Ser.Pop()
	trace2("Before entering expression")
	if object != nil {
		switch object.Type() {
		case env.IntegerType:
			stack.Push(es, object)
		case env.DecimalType:
			stack.Push(es, object)
		case env.StringType:
			stack.Push(es, object)
		case env.BlockType:
			stack.Push(es, object)
		case env.UriType:
			stack.Push(es, object)
		case env.EmailType:
			stack.Push(es, object)
		case env.WordType:
			rr := Eyr_EvalWord(es, object.(env.Word), nil, false, stack)
			return rr
		case env.OpwordType: // + and other operators are basically opwords too
			rr := Eyr_EvalWord(es, object.(env.Opword), nil, false, stack)
			return rr
		case env.CPathType:
			rr := Eyr_EvalWord(es, object.(env.CPath), nil, false, stack)
			return rr
		case env.LSetwordType:
			print(stack)
			rr := Eyr_EvalLSetword(es, object.(env.LSetword), nil, false, stack)
			return rr
		case env.BuiltinType:
			return Eyr_EvalObject(es, object, nil, false, nil, stack, false) //ww0128a *
		default:
			es.ErrorFlag = true
			es.Res = env.NewError("Not known type for Eyr")
		}
	} else {
		es.ErrorFlag = true
		es.Res = env.NewError("Not known type (nil)")
	}

	return es
}

func Eyr_EvalBlock(ps *env.ProgramState, stack *env.EyrStack, full bool) *env.ProgramState {
	for ps.Ser.Pos() < ps.Ser.Len() {
		ps = Eyr_EvalExpression(ps, stack)
		if checkFlagsAfterBlock(ps, 101) {
			return ps
		}
		if ps.ReturnFlag || ps.ErrorFlag {
			return ps
		}
	}
	if stack.I > 1 && full {
		ps.Res = *env.NewBlock(*env.NewTSeries(stack.D[0:stack.I]))
	} else if stack.I == 1 || (!full && !stack.IsEmpty()) {
		ps.Res = stack.Pop(ps)
	} else if stack.IsEmpty() {
		ps.Res = env.Void{}
	}

	return ps
}

var Builtins_eyr = map[string]*env.Builtin{

	"eyr": {
		Argsn: 1,
		Doc:   "Evaluates Rye block as Eyr (postfix) stack based code.",
		Fn: func(ps *env.ProgramState, arg0 env.Object, arg1 env.Object, arg2 env.Object, arg3 env.Object, arg4 env.Object) env.Object {
			switch bloc := arg0.(type) {
			case env.Block:
				stack := env.NewEyrStack()
				ser := ps.Ser
				ps.Ser = bloc.Series
				ps.Dialect = env.EyrDialect
				Eyr_EvalBlock(ps, stack, false)
				ps.Ser = ser
				return ps.Res
			default:
				return MakeArgError(ps, 1, []env.Type{env.BlockType}, "eyr")
			}
		},
	},

	"eyr\\full": {
		Argsn: 1,
		Doc:   "Evaluates Rye block as Eyr (postfix) stack based code.",
		Fn: func(ps *env.ProgramState, arg0 env.Object, arg1 env.Object, arg2 env.Object, arg3 env.Object, arg4 env.Object) env.Object {
			switch bloc := arg0.(type) {
			case env.Block:
				stack := env.NewEyrStack()
				ser := ps.Ser
				ps.Ser = bloc.Series
				ps.Dialect = env.EyrDialect
				Eyr_EvalBlock(ps, stack, true)
				ps.Ser = ser
				return ps.Res
			default:
				return MakeArgError(ps, 1, []env.Type{env.BlockType}, "eyr\\full")
			}
		},
	},

	"eyr\\loop": {
		Argsn: 2,
		Doc:   "Evaluates Rye block in loop as Eyr code (postfix stack based) N times.",
		Fn: func(ps *env.ProgramState, arg0 env.Object, arg1 env.Object, arg2 env.Object, arg3 env.Object, arg4 env.Object) env.Object {
			switch cond := arg0.(type) {
			case env.Integer:
				switch bloc := arg1.(type) {
				case env.Block:
					ps.Dialect = env.EyrDialect
					ser := ps.Ser
					ps.Ser = bloc.Series
					stack := env.NewEyrStack()
					for i := 0; int64(i) < cond.Value; i++ {
						ps = Eyr_EvalBlock(ps, stack, false)
						ps.Ser.Reset()
					}
					ps.Ser = ser
					return ps.Res
				default:
					return MakeArgError(ps, 2, []env.Type{env.BlockType}, "eyr\\loop")
				}
			default:
				return MakeArgError(ps, 1, []env.Type{env.IntegerType}, "eyr\\loop")
			}
		},
	},
	"to-eyr": {
		Argsn: 1,
		Doc:   "Evaluates Rye block as Eyr (postfix) stack based code.",
		Fn: func(ps *env.ProgramState, arg0 env.Object, arg1 env.Object, arg2 env.Object, arg3 env.Object, arg4 env.Object) env.Object {
			switch bloc := arg0.(type) {
			case env.Block:
				eBlock := env.NewBlock(*env.NewTSeries(make([]env.Object, 0)))
				CompileRyeToEyr(&bloc, ps, eBlock)
				return *eBlock
			default:
				return MakeArgError(ps, 1, []env.Type{env.BlockType}, "eyr")
			}
		},
	},
}

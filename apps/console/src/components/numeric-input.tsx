import * as React from "react";

import { Input } from "@/components/ui/input";

type NumericInputProps = Omit<
  React.ComponentProps<typeof Input>,
  "type" | "inputMode" | "pattern"
> & {
  allowNegative?: boolean;
};

/**
 * NumericInput keeps decimal values as strings so financial precision is not
 * lost in the browser while rejecting non-numeric keystrokes.
 */
export function NumericInput({
  allowNegative = false,
  min,
  step = "any",
  onChange,
  ...props
}: NumericInputProps) {
  const validValue = allowNegative ? /^-?\d*(?:\.\d*)?$/ : /^\d*(?:\.\d*)?$/;

  return (
    <Input
      {...props}
      type="number"
      inputMode="decimal"
      min={min ?? (allowNegative ? undefined : "0")}
      step={step}
      onChange={(event) => {
        if (validValue.test(event.target.value)) onChange?.(event);
      }}
    />
  );
}

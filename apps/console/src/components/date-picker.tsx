import { format } from "date-fns";
import { ChevronDownIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Calendar } from "@/components/ui/calendar";
import { Input } from "@/components/ui/input";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { cn } from "@/lib/utils";

type DatePickerProps = {
  value: string;
  onValueChange: (value: string) => void;
  placeholder?: string;
  min?: string;
  required?: boolean;
  className?: string;
};

const parseDate = (value: string) => {
  if (!value) return undefined;
  const [year, month, day] = value.split("-").map(Number);
  if (!year || !month || !day) return undefined;
  return new Date(year, month - 1, day);
};

const dateValue = (date: Date) =>
  `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`;

export function DatePicker({
  value,
  onValueChange,
  placeholder = "Pick a date",
  min,
  required,
  className,
}: DatePickerProps) {
  const selected = parseDate(value);
  const minimum = min ? parseDate(min.slice(0, 10)) : undefined;

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button
          type="button"
          variant="outline"
          data-empty={!selected}
          aria-required={required}
          className={cn(
            "w-full justify-between text-left font-normal data-[empty=true]:text-muted-foreground",
            className,
          )}
        >
          {selected ? format(selected, "PPP") : <span>{placeholder}</span>}
          <ChevronDownIcon data-icon="inline-end" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-auto p-0" align="start">
        <Calendar
          mode="single"
          selected={selected}
          defaultMonth={selected ?? minimum}
          disabled={minimum ? { before: minimum } : undefined}
          onSelect={(date) => {
            if (!date) return;
            onValueChange(dateValue(date));
          }}
        />
      </PopoverContent>
    </Popover>
  );
}

type DateTimePickerProps = Omit<DatePickerProps, "value" | "onValueChange"> & {
  value: string;
  onValueChange: (value: string) => void;
};

export function DateTimePicker({
  value,
  onValueChange,
  placeholder = "Pick a date and time",
  min,
  required,
  className,
}: DateTimePickerProps) {
  const [date = "", time = ""] = value.split("T");

  return (
    <div
      className={cn("grid gap-2 sm:grid-cols-[minmax(0,1fr)_8rem]", className)}
    >
      <DatePicker
        value={date}
        min={min}
        required={required}
        placeholder={placeholder}
        onValueChange={(nextDate) =>
          onValueChange(`${nextDate}T${time || "00:00"}`)
        }
      />
      <Input
        type="time"
        required={required}
        value={time}
        aria-label="Time"
        onChange={(event) => {
          const nextDate = date || dateValue(new Date());
          onValueChange(`${nextDate}T${event.target.value}`);
        }}
      />
    </div>
  );
}

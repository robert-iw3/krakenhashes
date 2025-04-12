import React from 'react';
import { Controller, FieldValues, UseControllerProps, ControllerRenderProps } from 'react-hook-form';
import {
  FormControl,
  InputLabel,
  CircularProgress,
  Box,
  TextField,
  Autocomplete
} from '@mui/material';
import { useQuery } from '@tanstack/react-query';
import { api } from '../../services/api';
import { AxiosResponse } from 'axios';

interface HashType {
  id: number;
  name: string;
  // Add other relevant fields from API if necessary
}

interface HashTypeApiResponse {
  data: HashType[];
  // Add other potential response fields like total_count, etc.
}

interface HashTypeSelectProps {
  control: any;
  name: string;
  label: string;
  error?: boolean;
  helperText?: string;
}

export default function HashTypeSelect({
  control,
  name,
  label,
  error,
  helperText
}: HashTypeSelectProps) {
  const { data: hashTypes = [], isLoading, isError } = useQuery<HashType[], Error>({
    queryKey: ['hashTypes'],
    queryFn: async () => {
      try {
        const response = await api.get<HashType[] | HashTypeApiResponse>('/api/hashtypes');
        console.debug("Fetched hash types raw data:", response.data);
        const dataArray = Array.isArray(response.data) ? response.data : response.data?.data;
        const result = Array.isArray(dataArray) ? dataArray : [];
        console.debug("Processed hash types for dropdown:", result);
        return result;
      } catch (error) {
        console.error("Failed to fetch hash types:", error);
        return [];
      }
    },
  });

  return (
    <Controller
      name={name}
      control={control}
      render={({ field }: { field: ControllerRenderProps<FieldValues, typeof name> }) => {
        const selectedOption = Array.isArray(hashTypes)
          ? hashTypes.find(option => option && option.id === field.value) || null
          : null;

        return (
          <Autocomplete
            options={hashTypes}
            getOptionLabel={(option) => `${option.id} - ${option.name}` || ''}
            isOptionEqualToValue={(option, value) => option.id === value?.id}
            value={selectedOption}
            loading={isLoading}
            disabled={isLoading}
            onChange={(_, newValue) => {
              field.onChange(newValue?.id ?? null);
            }}
            renderInput={(params) => (
              <TextField
                {...params}
                label={label}
                variant="outlined"
                margin="normal"
                error={error || isError}
                helperText={isError ? 'Failed to load hash types' : helperText}
                InputProps={{
                  ...params.InputProps,
                  endAdornment: (
                    <React.Fragment>
                      {isLoading ? <CircularProgress color="inherit" size={20} /> : null}
                      {params.InputProps.endAdornment}
                    </React.Fragment>
                  ),
                }}
              />
            )}
            ListboxProps={{ style: { maxHeight: 200 } }}
          />
        );
      }}
    />
  );
}
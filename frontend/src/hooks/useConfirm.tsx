import React, { useState, useCallback } from 'react';
import { 
  Dialog, 
  DialogActions, 
  DialogContent, 
  DialogContentText, 
  DialogTitle, 
  Button 
} from '@mui/material';

interface ConfirmDialogState {
  isOpen: boolean;
  title: string;
  message: string;
  resolve: ((value: boolean) => void) | null;
}

const initialState: ConfirmDialogState = {
  isOpen: false,
  title: '',
  message: '',
  resolve: null,
};

export const useConfirm = () => {
  const [dialogState, setDialogState] = useState<ConfirmDialogState>(initialState);

  const handleClose = useCallback(() => {
    if (dialogState.resolve) {
      dialogState.resolve(false);
    }
    setDialogState(initialState);
  }, [dialogState]);

  const handleConfirm = useCallback(() => {
    if (dialogState.resolve) {
      dialogState.resolve(true);
    }
    setDialogState(initialState);
  }, [dialogState]);

  const showConfirm = useCallback((title: string, message: string): Promise<boolean> => {
    return new Promise<boolean>((resolve) => {
      setDialogState({
        isOpen: true,
        title,
        message,
        resolve,
      });
    });
  }, []);

  const ConfirmDialog = useCallback(() => (
    <Dialog
      open={dialogState.isOpen}
      onClose={handleClose}
      aria-labelledby="confirm-dialog-title"
      aria-describedby="confirm-dialog-description"
    >
      <DialogTitle id="confirm-dialog-title">{dialogState.title}</DialogTitle>
      <DialogContent>
        <DialogContentText id="confirm-dialog-description">
          {dialogState.message}
        </DialogContentText>
      </DialogContent>
      <DialogActions>
        <Button onClick={handleClose} color="primary">
          Cancel
        </Button>
        <Button onClick={handleConfirm} color="primary" variant="contained" autoFocus>
          Confirm
        </Button>
      </DialogActions>
    </Dialog>
  ), [dialogState, handleClose, handleConfirm]);

  return { showConfirm, ConfirmDialog };
}; 
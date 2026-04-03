import { createSlice } from "@reduxjs/toolkit";

interface ResourcesUiStateInterface {
  selectedCategory: string;
  searchQuery?: string;
}

const initialState: ResourcesUiStateInterface = {
  selectedCategory: "all",
  searchQuery: "",
};

const ResourcesUiState = createSlice({
  name: "resourcesUi",
  initialState,
  reducers: {
    setCategory: (state, action) => {
      state.selectedCategory = action.payload;
    },
    setSearchQuery: (state, action) => {
      state.searchQuery = action.payload;
    },
  },
});

export const { setCategory, setSearchQuery } = ResourcesUiState.actions;

export default ResourcesUiState.reducer;
